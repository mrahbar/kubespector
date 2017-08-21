package pkg

import (
	"bytes"
	"fmt"
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

func ClusterStatus(config types.Config, opts *types.ClusterStatusOpts) {
	clusterStatusOpts = opts

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

	util.PrettyPrint("Performing status checks %s for groups: %v", strings.Join(clusterStatusChecks, ","), strings.Join(groups, " "))

	for _, element := range groups {
		if element != types.KUBERNETES_GROUPNAME {
			group := util.FindGroupByName(config.ClusterGroups, element)
			if group.Nodes != nil {
				getNodesStats(config.Ssh, element, group.Nodes)

				if util.ElementInArray(clusterStatusChecks, types.SERVICES_CHECKNAME) {
					checkServiceStatus(config.Ssh, element, group.Services, group.Nodes)
				}

				if util.ElementInArray(clusterStatusChecks, types.CONTAINERS_CHECKNAME) {
					checkContainerStatus(config.Ssh, element, group.Containers, group.Nodes)
				}

				if util.ElementInArray(clusterStatusChecks, types.CERTIFICATES_CHECKNAME) {
					checkCertificatesExpiration(config.Ssh, element, group.Certificates, group.Nodes)
				}

				if util.ElementInArray(clusterStatusChecks, types.DISKUSAGE_CHECKNAME) {
					checkDiskStatus(config.Ssh, element, group.DiskUsage, group.Nodes)
				}
			} else {
				util.PrettyPrintErr("No Nodes found for group: %s", element)
			}
		} else {
			group := util.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)
			checkKubernetesStatus(config.Ssh, element, config.Kubernetes.Resources, group.Nodes)
		}
	}
}

func getNodesStats(sshOpts types.SSHConfig, element string, nodes []types.Node) {
	util.PrintHeader(fmt.Sprintf("Retrieving node stats of group [%s] ", element), '=')

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			util.PrettyPrintErr("Current node %q has no valid address", node)
			break
		}

		util.PrettyNewLine()
		util.PrettyPrint("On node %s:", util.ToNodeLabel(node))

		sshOut, err := ssh.PerformCmd(sshOpts, node, "cat /proc/uptime", clusterStatusOpts.Debug)
		if err != nil {
			util.PrettyPrintWarn("Could not get uptime for: %s", err)
		} else {
			parts := strings.Fields(sshOut.Stdout)
			if len(parts) == 2 {
				var upsecs float64
				upsecs, err = strconv.ParseFloat(parts[0], 64)
				if err != nil {
					util.PrettyPrintWarn("Could not parse uptime: %s", err)
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
					util.PrettyPrintOk("Uptime %s", uptimeFormated)
				}
			}
		}

		sshOut, err = ssh.PerformCmd(sshOpts, node, "/bin/cat /proc/loadavg", clusterStatusOpts.Debug)
		if err != nil {
			util.PrettyPrintWarn("Could not get load statistics: %s", err)
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

					util.PrettyPrintOk(strings.TrimSpace(loadMsg))
				}
			}
		}
	}
}

func checkServiceStatus(sshOpts types.SSHConfig, element string, services []string, nodes []types.Node) {
	util.PrintHeader(fmt.Sprintf("Checking service status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		util.PrettyPrintSkipped("No host configured for [%s]", element)
		return
	}
	if services == nil || len(services) == 0 {
		util.PrettyPrintSkipped("No services configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			util.PrettyPrintErr("Current node %q has no valid address", node)
			break
		}

		util.PrettyNewLine()
		util.PrettyPrint("On node %s:", util.ToNodeLabel(node))

		for _, service := range services {
			sshOut, err := ssh.PerformCmd(sshOpts, node,
				fmt.Sprintf("systemctl is-active %s", service), clusterStatusOpts.Debug)

			if err != nil {
				util.PrettyPrintErr("Error checking status of %s: %s", service, err)
			} else {
				result := sshOut.Stdout
				if result == "active" {
					util.PrettyPrintOk("Service %s is active", service)
				} else if result == "activating" || result == "inactive" {
					util.PrettyPrintWarn("Service %s is %s", service, result)
				} else if result == "failed" {
					util.PrettyPrintErr("Service %s is failed", service)
				} else {
					util.PrettyPrintUnknown("Service %s is unknown state: %s", service, result)
				}
			}
		}
	}
}

func checkContainerStatus(sshOpts types.SSHConfig, element string, containers []string, nodes []types.Node) {
	util.PrintHeader(fmt.Sprintf("Checking container status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		util.PrettyPrintSkipped("No host configured for [%s]", element)
		return
	}
	if containers == nil || len(containers) == 0 {
		util.PrettyPrintSkipped("No containers configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			util.PrettyPrintErr("Current node %q has no valid address", node)
			break
		}

		util.PrettyNewLine()
		util.PrettyPrint("On node %s:", util.ToNodeLabel(node))

		for _, container := range containers {
			sshOut, err := ssh.PerformCmd(sshOpts, node,
				fmt.Sprintf("sudo bash -c 'docker ps -a -q --latest -f name=%s* | xargs --no-run-if-empty docker inspect -f '{{.State.Status}}''", container), clusterStatusOpts.Debug)

			if err != nil {
				util.PrettyPrintErr("Error checking status of %s: %s", container, err)
			} else {
				result := sshOut.Stdout
				if result == "running" {
					util.PrettyPrintOk("Container %s is running", container)
				} else if result == "created" || result == "paused" || result == "restarting" {
					util.PrettyPrintWarn("Container %s is %s", container, result)
				} else if result == "exited" || result == "removing" || result == "dead" {
					util.PrettyPrintErr("Container %s is %s", container, result)
				} else {
					util.PrettyPrintIgnored("Container %s not found or in unknown state: %s", container, result)
				}
			}
		}
	}
}

func checkCertificatesExpiration(sshOpts types.SSHConfig, element string, certificates []string, nodes []types.Node) {
	util.PrintHeader(fmt.Sprintf("Checking certificate status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		util.PrettyPrintSkipped("No host configured for [%s]", element)
		return
	}
	if certificates == nil || len(certificates) == 0 {
		util.PrettyPrintSkipped("No certificates configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			util.PrettyPrintErr("Current node %q has no valid address", node)
			break
		}

		util.PrettyNewLine()
		util.PrettyPrint("On node %s:", util.ToNodeLabel(node))

		for _, cert := range certificates {
			cert = parseTemplate(cert, node, clusterStatusOpts.Debug)
			sshOut, err := ssh.PerformCmd(sshOpts, node,
				fmt.Sprintf("sudo bash -c 'openssl x509 -enddate -noout -in %s | cut -d= -f 2'", cert), clusterStatusOpts.Debug)

			if err != nil {
				util.PrettyPrintErr("Error checking expiration of %s: %s", cert, err)
			} else {
				_, err = ssh.PerformCmd(sshOpts, node,
					fmt.Sprintf("sudo openssl x509 -checkend 86400 -noout -in %s", cert), clusterStatusOpts.Debug)

				if err == nil {
					util.PrettyPrintOk("Certificate %s is valid until %s", cert, sshOut.Stdout)
				} else {
					util.PrettyPrintWarn("Certificate %s is only valid until %s", cert, sshOut.Stdout)
				}
			}
		}
	}
}

func parseTemplate(value string, node types.Node, debug bool) string {
	if strings.Contains(value, leftTemplateDelim) && strings.Contains(value, rightTemplateDelim) {
		if debug {
			util.PrettyPrintDebug("Value containts templating. Parsing: %s", value)
		}

		t := template.New("Template")
		t, err := t.Parse(value)
		if err != nil {
			if debug {
				util.PrettyPrintDebug("Error parsing template: %s", err)
			}
			return value
		}

		var tplResult bytes.Buffer
		err = t.Execute(&tplResult, node)
		if err != nil {
			if debug {
				util.PrettyPrintDebug("Error executing template: %s", err)
			}
		} else {
			value = tplResult.String()
			if debug {
				util.PrettyPrintDebug("Template executed successfully: %s", value)
			}
			return value
		}
	}

	if debug {
		util.PrettyPrintDebug("Value does not containts templating: %s", value)
	}

	return value
}

func checkDiskStatus(sshOpts types.SSHConfig, element string, diskSpace types.DiskUsage, nodes []types.Node) {
	util.PrintHeader(fmt.Sprintf("Checking disk status of group [%s] ", element), '-')
	if nodes == nil || len(nodes) == 0 {
		util.PrettyPrintSkipped("No host configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			util.PrettyPrintErr("Current node %q has no valid address", node)
			break
		}

		util.PrettyNewLine()
		util.PrettyPrint("On node %s:", util.ToNodeLabel(node))

		spacesRegex := regexp.MustCompile("\\s+")
		if len(diskSpace.FileSystemUsage) > 0 {
			for _, fsUsage := range diskSpace.FileSystemUsage {
				sshOut, err := ssh.PerformCmd(sshOpts, node,
					fmt.Sprintf("sudo df -h | grep %s", fsUsage), clusterStatusOpts.Debug)

				if err != nil {
					util.PrettyPrintErr("Error estimating file system usage for %s: %s", fsUsage, err)
				} else {
					splits := spacesRegex.Split(sshOut.Stdout, 6)
					fsUsed := splits[2]
					fsAvail := splits[3]
					fsUsePercent := strings.Replace(splits[4], "%", "", 1)
					fsUsePercentVal, err := strconv.Atoi(fsUsePercent)

					if err != nil {
						util.PrettyPrintErr("Error determining file system usage percent for %s: %s", fsUsage, err)
					} else {
						if fsUsePercentVal < 65 {
							util.PrettyPrintOk("File system usage of %s amounts to - Used: %s Available: %s (%s%%)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else if fsUsePercentVal < 85 {
							util.PrettyPrintWarn("File system usage of %s amounts to - Used: %s Available: %s (%s%%)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else {
							util.PrettyPrintErr("File system usage of %s amounts to - Used: %s Available: %s (%s%%)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						}
					}
				}
			}
		}

		if len(diskSpace.DirectoryUsage) > 0 {
			for _, dirUsage := range diskSpace.DirectoryUsage {
				sshOut, err := ssh.PerformCmd(sshOpts, node,
					fmt.Sprintf("sudo du -h -d 0 --exclude=/proc --exclude=/run %s | grep %s", dirUsage, dirUsage),
					clusterStatusOpts.Debug)

				if err != nil {
					util.PrettyPrintErr("Error estimating directory usage for %s: %s", dirUsage, err)
				} else {
					splits := spacesRegex.Split(sshOut.Stdout, 2)
					dirUse := splits[0]

					util.PrettyPrintOk("Directory usage of %s amounts to %s", dirUsage, dirUse)
				}
			}
		}
	}
}

func checkKubernetesStatus(sshOpts types.SSHConfig, element string,
	resources []types.KubernetesResource, nodes []types.Node) {
	util.PrintHeader(fmt.Sprintf("Checking status of [%s] ", element), '=')

	if nodes == nil || len(nodes) == 0 {
		util.PrettyPrintErr("No host configured for [%s]", element)
		os.Exit(1)
	}
	if resources == nil || len(resources) == 0 {
		util.PrettyPrintErr("No resources configured for [%s]", element)
		os.Exit(1)
	}

	node := ssh.GetFirstAccessibleNode(sshOpts, nodes, clusterStatusOpts.Debug)

	if !util.IsNodeAddressValid(node) {
		util.PrettyPrintErr("No master available for Kubernetes status check")
		os.Exit(1)
	}

	util.PrettyPrint("Running kubectl on node %s\n", util.ToNodeLabel(node))

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

		util.PrettyPrint(msg + namespace_msg + ":")
		sshOut, err := ssh.PerformCmd(sshOpts, node, command, clusterStatusOpts.Debug)
		util.PrettyNewLine()

		if err != nil {
			util.PrettyPrintErr("Error checking %s%s: %s", resource.Type, namespace_msg, err)
		} else {
			util.PrettyPrintOk(sshOut.Stdout)
		}
		util.PrettyNewLine()
	}
}
