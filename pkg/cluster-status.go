package pkg

import (
	"bytes"
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	leftTemplateDelim  = "{{"
	rightTemplateDelim = "}}"
)

var clusterStatusChecks = []string{types.SERVICES_CHECKNAME, types.CONTAINERS_CHECKNAME, types.CERTIFICATES_CHECKNAME, types.DISKUSAGE_CHECKNAME}
var clusterStatusOpts = &types.ClusterStatusOpts{}

func ClusterStatus(cmdParams *types.CommandParams) {
	initParams(cmdParams)
	clusterStatusOpts = cmdParams.Opts.(*types.ClusterStatusOpts)

	var groups = []string{}
	if clusterStatusOpts.Groups != "" {
		groups = strings.Split(clusterStatusOpts.Groups, ",")
	} else {
		for _, group := range config.ClusterGroups {
			groups = append(groups, group.Name)
		}
		groups = append(groups, types.KUBERNETES_GROUPNAME)
	}

	if clusterStatusOpts.Checks != "" {
		clusterStatusChecks = strings.Split(clusterStatusOpts.Checks, ",")
	}

	printer.Print("Performing status checks %s for groups: %v",
		strings.Join(clusterStatusChecks, ","), strings.Join(groups, " "))

	cmdExecutor := &ssh.CommandExecutor{
		SshOpts: config.Ssh,
		Printer: printer,
	}

	for _, element := range groups {
		if element != types.KUBERNETES_GROUPNAME {
			group := util.FindGroupByName(config.ClusterGroups, element)
			if group.Nodes != nil {
				getNodesStats(cmdExecutor, element, group.Nodes)

				if util.ElementInArray(clusterStatusChecks, types.SERVICES_CHECKNAME) {
					checkServiceStatus(cmdExecutor, element, group.Services, group.Nodes)
				}

				if util.ElementInArray(clusterStatusChecks, types.CONTAINERS_CHECKNAME) {
					checkContainerStatus(cmdExecutor, element, group.Containers, group.Nodes)
				}

				if util.ElementInArray(clusterStatusChecks, types.CERTIFICATES_CHECKNAME) {
					checkCertificatesExpiration(cmdExecutor, element, group.Certificates, group.Nodes)
				}

				if util.ElementInArray(clusterStatusChecks, types.DISKUSAGE_CHECKNAME) {
					checkDiskStatus(cmdExecutor, element, group.DiskUsage, group.Nodes)
				}
			} else {
				printer.PrintErr("No Nodes found for group: %s", element)
			}
		} else {
			group := util.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)
			checkKubernetesStatus(cmdExecutor, element, config.Kubernetes.Resources, group.Nodes)
		}
	}
}

func getNodesStats(cmdExecutor *ssh.CommandExecutor, element string, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Retrieving node stats of group [%s] ", element), '=')

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			printer.PrintErr("Current node %q has no valid address", node)
			break
		}

		integration.PrettyNewLine()
		printer.Print("On node %s:", util.ToNodeLabel(node))
		cmdExecutor.Node = node

		sshOut, err := cmdExecutor.PerformCmd("cat /proc/uptime")
		if err != nil {
			printer.PrintWarn("Could not get uptime for: %s", err)
		} else {
			parts := strings.Fields(sshOut.Stdout)
			if len(parts) == 2 {
				var upsecs float64
				upsecs, err = strconv.ParseFloat(parts[0], 64)
				if err != nil {
					printer.PrintWarn("Could not parse uptime: %s", err)
				} else {
					dur := time.Duration(upsecs * 1e9)
					dur = dur - (dur % time.Second)
					var days int
					for dur.Hours() > 24.0 {
						days++
						dur -= 24 * time.Hour
					}
					s1 := dur.String()
					uptimeFormated := ""
					if days > 0 {
						uptimeFormated = fmt.Sprintf("%dd ", days)
					}
					for _, ch := range s1 {
						uptimeFormated += string(ch)
						if ch == 'h' || ch == 'm' {
							uptimeFormated += " "
						}
					}
					printer.PrintOk("Uptime %s", uptimeFormated)
				}
			}
		}

		sshOut, err = cmdExecutor.PerformCmd("/bin/cat /proc/loadavg")
		if err != nil {
			printer.PrintWarn("Could not get load statistics: %s", err)
		} else {
			parts := strings.Fields(sshOut.Stdout)
			if len(parts) == 5 {
				loadMsg := fmt.Sprintf("Load periods in minutes - 1: %s - 5: %s - 10: %s\n", parts[0], parts[1], parts[2])

				if i := strings.Index(parts[3], "/"); i != -1 {
					runningProcs := parts[3][0:i]
					totalProcs := "-"
					if i+1 < len(parts[3]) {
						totalProcs = parts[3][i+1:]
					}
					loadMsg = fmt.Sprintf("%sNumber of processes: currently running %s - total %s", loadMsg, runningProcs, totalProcs)

					printer.PrintOk(strings.TrimSpace(loadMsg))
				}
			}
		}
	}
}

func checkServiceStatus(cmdExecutor *ssh.CommandExecutor, element string, services []string, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Checking service status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		printer.PrintSkipped("No host configured for [%s]", element)
		return
	}
	if services == nil || len(services) == 0 {
		printer.PrintSkipped("No services configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			printer.PrintErr("Current node %q has no valid address", node)
			break
		}

		integration.PrettyNewLine()
		printer.Print("On node %s:", util.ToNodeLabel(node))

		for _, service := range services {
			sshOut, err := cmdExecutor.PerformCmd(fmt.Sprintf("systemctl is-active %s", service))

			if err != nil {
				printer.PrintErr("Error checking status of %s: %s", service, err)
			} else {
				result := sshOut.Stdout
				if result == "active" {
					printer.PrintOk("Service %s is active", service)
				} else if result == "activating" || result == "inactive" {
					printer.PrintWarn("Service %s is %s", service, result)
				} else if result == "failed" {
					printer.PrintErr("Service %s is failed", service)
				} else {
					printer.PrintUnknown("Service %s is unknown state: %s", service, result)
				}
			}
		}
	}
}

func checkContainerStatus(cmdExecutor *ssh.CommandExecutor, element string, containers []string, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Checking container status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		printer.PrintSkipped("No host configured for [%s]", element)
		return
	}
	if containers == nil || len(containers) == 0 {
		printer.PrintSkipped("No containers configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			printer.PrintErr("Current node %q has no valid address", node)
			break
		}

		integration.PrettyNewLine()
		printer.Print("On node %s:", util.ToNodeLabel(node))

		for _, container := range containers {
			cmd := fmt.Sprintf("sudo bash -c 'docker ps -a -q --latest -f name=%s* | xargs --no-run-if-empty docker inspect -f '{{.State.Status}}''", container)
			sshOut, err := cmdExecutor.PerformCmd(cmd)

			if err != nil {
				printer.PrintErr("Error checking status of %s: %s", container, err)
			} else {
				result := sshOut.Stdout
				if result == "running" {
					printer.PrintOk("Container %s is running", container)
				} else if result == "created" || result == "paused" || result == "restarting" {
					printer.PrintWarn("Container %s is %s", container, result)
				} else if result == "exited" || result == "removing" || result == "dead" {
					printer.PrintErr("Container %s is %s", container, result)
				} else {
					printer.PrintIgnored("Container %s not found or in unknown state: %s", container, result)
				}
			}
		}
	}
}

func checkCertificatesExpiration(cmdExecutor *ssh.CommandExecutor, element string, certificates []string, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Checking certificate status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		printer.PrintSkipped("No host configured for [%s]", element)
		return
	}
	if certificates == nil || len(certificates) == 0 {
		printer.PrintSkipped("No certificates configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			printer.PrintErr("Current node %q has no valid address", node)
			break
		}

		integration.PrettyNewLine()
		printer.Print("On node %s:", util.ToNodeLabel(node))

		for _, cert := range certificates {
			cert = parseTemplate(cert, node)
			sshOut, err := cmdExecutor.PerformCmd(fmt.Sprintf("sudo bash -c 'openssl x509 -enddate -noout -in %s | cut -d= -f 2'", cert))

			if err != nil {
				printer.PrintErr("Error checking expiration of %s: %s", cert, err)
			} else {
				//TODO this is not robust enough
				_, err = cmdExecutor.PerformCmd(fmt.Sprintf("sudo openssl x509 -checkend 86400 -noout -in %s", cert))

				if err == nil {
					printer.PrintOk("Certificate %s is valid until %s", cert, sshOut.Stdout)
				} else {
					printer.PrintWarn("Certificate %s is only valid until %s", cert, sshOut.Stdout)
				}
			}
		}
	}
}

func parseTemplate(value string, node types.Node) string {
	if strings.Contains(value, leftTemplateDelim) && strings.Contains(value, rightTemplateDelim) {
		printer.PrintDebug("Value containts templating. Parsing: %s", value)

		t := template.New("Template")
		t, err := t.Parse(value)
		if err != nil {
			printer.PrintDebug("Error parsing template: %s", err)
			return value
		}

		var tplResult bytes.Buffer
		err = t.Execute(&tplResult, node)
		if err != nil {
			printer.PrintDebug("Error executing template: %s", err)
		} else {
			value = tplResult.String()
			printer.PrintDebug("Template executed successfully: %s", value)
			return value
		}
	}

	printer.PrintDebug("Value does not containts templating: %s", value)

	return value
}

func checkDiskStatus(cmdExecutor *ssh.CommandExecutor, element string, diskSpace types.DiskUsage, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Checking disk status of group [%s] ", element), '-')
	if nodes == nil || len(nodes) == 0 {
		printer.PrintSkipped("No host configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			printer.PrintErr("Current node %q has no valid address", node)
			break
		}

		integration.PrettyNewLine()
		printer.Print("On node %s:", util.ToNodeLabel(node))

		spacesRegex := regexp.MustCompile("\\s+")
		if len(diskSpace.FileSystemUsage) > 0 {
			for _, fsUsage := range diskSpace.FileSystemUsage {
				sshOut, err := cmdExecutor.PerformCmd(fmt.Sprintf("sudo df -h | grep %s", fsUsage))

				if err != nil {
					printer.PrintErr("Error estimating file system usage for %s: %s", fsUsage, err)
				} else {
					splits := spacesRegex.Split(sshOut.Stdout, 6)
					fsUsed := splits[2]
					fsAvail := splits[3]
					fsUsePercent := strings.Replace(splits[4], "%", "", 1)
					fsUsePercentVal, err := strconv.Atoi(fsUsePercent)

					if err != nil {
						printer.PrintErr("Error determining file system usage percent for %s: %s", fsUsage, err)
					} else {
						if fsUsePercentVal < 65 {
							printer.PrintOk("File system usage of %s amounts to - Used: %s Available: %s (%s%%)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else if fsUsePercentVal < 85 {
							printer.PrintWarn("File system usage of %s amounts to - Used: %s Available: %s (%s%%)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else {
							printer.PrintErr("File system usage of %s amounts to - Used: %s Available: %s (%s%%)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						}
					}
				}
			}
		}

		if len(diskSpace.DirectoryUsage) > 0 {
			for _, dirUsage := range diskSpace.DirectoryUsage {
				cmd := fmt.Sprintf("sudo du -h -d 0 --exclude=/proc --exclude=/run %s | grep %s", dirUsage, dirUsage)
				sshOut, err := cmdExecutor.PerformCmd(cmd)

				if err != nil {
					printer.PrintErr("Error estimating directory usage for %s: %s", dirUsage, err)
				} else {
					splits := spacesRegex.Split(sshOut.Stdout, 2)
					dirUse := splits[0]

					printer.PrintOk("Directory usage of %s amounts to %s", dirUsage, dirUse)
				}
			}
		}
	}
}

func checkKubernetesStatus(cmdExecutor *ssh.CommandExecutor, element string,
	resources []types.KubernetesResource, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Checking status of [%s] ", element), '=')

	if nodes == nil || len(nodes) == 0 {
		printer.PrintErr("No host configured for [%s]", element)
		os.Exit(1)
	}
	if resources == nil || len(resources) == 0 {
		printer.PrintErr("No resources configured for [%s]", element)
		os.Exit(1)
	}

	node := ssh.GetFirstAccessibleNode(sshOpts, nodes, cmdParams.Printer)

	if !util.IsNodeAddressValid(node) {
		printer.PrintErr("No master available for Kubernetes status check")
		os.Exit(1)
	}

	printer.Print("Running kubectl on node %s\n", util.ToNodeLabel(node))

	for _, resource := range resources {
		msg := fmt.Sprintf("Status of %s", resource.Type)
		namespace_msg := ""
		command := fmt.Sprintf("kubectl get %s", resource.Type)
		if resource.Namespace != "" {
			namespace_msg += " in namespace " + resource.Namespace
			command += " -n " + resource.Namespace
		}
		if resource.Wide {
			command += " -o wide"
		}

		printer.Print(msg + namespace_msg + ":")
		sshOut, err := cmdExecutor.PerformCmd(command)
		integration.PrettyNewLine()

		if err != nil {
			printer.PrintErr("Error checking %s%s: %s", resource.Type, namespace_msg, err)
		} else {
			printer.PrintOk(sshOut.Stdout)
		}
		integration.PrettyNewLine()
	}
}
