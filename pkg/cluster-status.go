package pkg

import (
	"bytes"
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/types"
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

	integration.PrettyPrint("Performing status checks %s for groups: %v\n", strings.Join(clusterStatusChecks, ","), strings.Join(groups, " "))

	for _, element := range groups {
		if element != types.KUBERNETES_GROUPNAME {
			group := integration.FindGroupByName(config.ClusterGroups, element)
			if group.Nodes != nil {
				getNodesStats(config.Ssh, element, group.Nodes)

				if integration.ElementInArray(clusterStatusChecks, types.SERVICES_CHECKNAME) {
					checkServiceStatus(config.Ssh, element, group.Services, group.Nodes)
				}

				if integration.ElementInArray(clusterStatusChecks, types.CONTAINERS_CHECKNAME) {
					checkContainerStatus(config.Ssh, element, group.Containers, group.Nodes)
				}

				if integration.ElementInArray(clusterStatusChecks, types.CERTIFICATES_CHECKNAME) {
					checkCertificatesExpiration(config.Ssh, element, group.Certificates, group.Nodes)
				}

				if integration.ElementInArray(clusterStatusChecks, types.DISKUSAGE_CHECKNAME) {
					checkDiskStatus(config.Ssh, element, group.DiskUsage, group.Nodes)
				}
			} else {
				integration.PrettyPrintErr("No Nodes found for group: %s", element)
			}
		} else {
			group := integration.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)
			checkKubernetesStatus(config.Ssh, element, config.Kubernetes.Resources, group.Nodes)
		}
	}
}

func getNodesStats(sshOpts types.SSHConfig, element string, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Retrieving node stats of group [%s] ", element), '=')

	for _, node := range nodes {
		if !integration.IsNodeAddressValid(node) {
			integration.PrettyPrintErr("Current node %q has no valid address", node)
			break
		}

		result, err := integration.PerformSSHCmd(sshOpts, node, "cat /proc/uptime", clusterStatusOpts.Debug)
		if err != nil {
			integration.PrettyPrintWarn("Could not get uptime for node %s: %s", integration.ToNodeLabel(node), err)
		} else {
			parts := strings.Fields(result)
			if len(parts) == 2 {
				var upsecs float64
				upsecs, err = strconv.ParseFloat(parts[0], 64)
				if err != nil {
					integration.PrettyPrintWarn("Could not parse uptime for node %s: %s", integration.ToNodeLabel(node), err)
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
					integration.PrettyPrintOk("Uptime of node %s: %s", integration.ToNodeLabel(node), uptimeFormated)
				}
			}
		}

		result, err = integration.PerformSSHCmd(sshOpts, node, "/bin/cat /proc/loadavg", clusterStatusOpts.Debug)
		if err != nil {
			integration.PrettyPrintWarn("Could not get load statistics for node %s: %s", integration.ToNodeLabel(node), err)
		} else {

			parts := strings.Fields(result)
			if len(parts) == 5 {
				loadMsg := fmt.Sprintf("Load average on node %s:\n", integration.ToNodeLabel(node))
				loadMsg = fmt.Sprintf("%s\tPeriods in minutes - 1: %s - 5:%s - 10:%s\n", loadMsg, parts[0], parts[1], parts[2])

				if i := strings.Index(parts[3], "/"); i != -1 {
					runningProcs := parts[3][0:i]
					totalProcs := "-"
					if i+1 < len(parts[3]) {
						totalProcs = parts[3][i+1:]
					}
					loadMsg = fmt.Sprintf("%s\tNumber of processes: currently running %s - total %s\n", loadMsg, runningProcs, totalProcs)

					integration.PrettyPrintOk(loadMsg)
				}
			}
		}
	}

	integration.PrettyPrint("\n")
}

func checkServiceStatus(sshOpts types.SSHConfig, element string, services []string, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Checking service status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		integration.PrettyPrintIgnored("No host configured for [%s]", element)
		return
	}
	if services == nil || len(services) == 0 {
		integration.PrettyPrintIgnored("No services configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !integration.IsNodeAddressValid(node) {
			integration.PrettyPrintErr("Current node %q has no valid address", node)
			break
		}

		integration.PrettyPrint("\nOn node %s:\n", integration.ToNodeLabel(node))

		for _, service := range services {
			result, err := integration.PerformSSHCmd(sshOpts, node,
				fmt.Sprintf("systemctl is-active %s", service), clusterStatusOpts.Debug)

			if err != nil {
				integration.PrettyPrintErr("Error checking status of %s: %s, %s", service, result, err)
			} else {
				if result == "active" {
					integration.PrettyPrintOk("Service %s is active", service)
				} else if result == "activating" || result == "inactive" {
					integration.PrettyPrintWarn("Service %s is %s", service, result)
				} else if result == "failed" {
					integration.PrettyPrintErr("Service %s is failed", service)
				} else {
					integration.PrettyPrintUnknown("Service %s is unknown state: %s", service, result)
				}
			}
		}
	}
}

func checkContainerStatus(sshOpts types.SSHConfig, element string, containers []string, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Checking container status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		integration.PrettyPrintIgnored("No host configured for [%s]", element)
		return
	}
	if containers == nil || len(containers) == 0 {
		integration.PrettyPrintIgnored("No containers configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !integration.IsNodeAddressValid(node) {
			integration.PrettyPrintErr("Current node %q has no valid address", node)
			break
		}

		integration.PrettyPrint("\nOn node %s:\n", integration.ToNodeLabel(node))

		for _, container := range containers {
			result, err := integration.PerformSSHCmd(sshOpts, node,
				fmt.Sprintf("bash -c 'docker ps -a --latest -f name=%s* -q | xargs --no-run-if-empty docker inspect -f '{{.State.Status}}''", container), clusterStatusOpts.Debug)

			if err != nil {
				integration.PrettyPrintErr("Error checking status of %s: %s, %s", container, result, err)
			} else {
				if result == "running" {
					integration.PrettyPrintOk("Container %s is running", container)
				} else if result == "created" || result == "paused" || result == "restarting" {
					integration.PrettyPrintWarn("Container %s is %s", container, result)
				} else if result == "exited" || result == "removing" || result == "dead" {
					integration.PrettyPrintErr("Container %s is %s", container, result)
				} else {
					integration.PrettyPrintSkipped("Container %s not found or in unknown state: %s", container, result)
				}
			}
		}
	}
}

func checkCertificatesExpiration(sshOpts types.SSHConfig, element string, certificates []string, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Checking certificate status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		integration.PrettyPrintIgnored("No host configured for [%s]", element)
		return
	}
	if certificates == nil || len(certificates) == 0 {
		integration.PrettyPrintIgnored("No certificates configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !integration.IsNodeAddressValid(node) {
			integration.PrettyPrintErr("Current node %q has no valid address", node)
			break
		}

		integration.PrettyPrint("\nOn node %s:\n", integration.ToNodeLabel(node))

		for _, cert := range certificates {
			cert = parseTemplate(cert, node, clusterStatusOpts.Debug)
			result, err := integration.PerformSSHCmd(sshOpts, node,
				fmt.Sprintf("bash -c 'openssl x509 -enddate -noout -in %s |cut -d= -f 2'", cert), clusterStatusOpts.Debug)

			if err != nil {
				integration.PrettyPrintErr("Error checking expiration of %s: %s, %s", cert, result, err)
			} else {
				_, err = integration.PerformSSHCmd(sshOpts, node,
					fmt.Sprintf("openssl x509 -checkend 86400 -noout -in %s", cert), clusterStatusOpts.Debug)

				if err == nil {
					integration.PrettyPrintOk("Certificate %s is valid until %s", cert, result)
				} else {
					integration.PrettyPrintWarn("Certificate %s is only valid until %s", cert, result)
				}
			}
		}
	}
}

func parseTemplate(value string, node types.Node, debug bool) string {
	if strings.Contains(value, leftTemplateDelim) && strings.Contains(value, rightTemplateDelim) {
		if debug {
			fmt.Printf("Value containts templating. Parsing: %s\n", value)
		}

		t := template.New("Template")
		t, err := t.Parse(value)
		if err != nil {
			if debug {
				fmt.Printf("Error parsing template: %s\n", err)
			}
			return value
		}

		var tplResult bytes.Buffer
		err = t.Execute(&tplResult, node)
		if err != nil {
			if debug {
				fmt.Printf("Error executing template: %s\n", err)
			}
		} else {
			value = tplResult.String()
			if debug {
				fmt.Printf("Template executed successfully: %s\n", value)
			}
			return value
		}
	}

	if debug {
		fmt.Printf("Value does not containts templating: %s\n", value)
	}

	return value
}

func checkDiskStatus(sshOpts types.SSHConfig, element string, diskSpace types.DiskUsage, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Checking disk status of group [%s] ", element), '-')
	if nodes == nil || len(nodes) == 0 {
		integration.PrettyPrintIgnored("No host configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !integration.IsNodeAddressValid(node) {
			integration.PrettyPrintErr("Current node %q has no valid address", node)
			break
		}

		integration.PrettyPrint("\nOn node %s:\n", integration.ToNodeLabel(node))

		if len(diskSpace.FileSystemUsage) > 0 {
			for _, fsUsage := range diskSpace.FileSystemUsage {
				result, err := integration.PerformSSHCmd(sshOpts, node,
					fmt.Sprintf("df -h | grep %s", fsUsage), clusterStatusOpts.Debug)

				if err != nil {
					integration.PrettyPrintErr("Error estimating file system usage for %s: %s, %s", fsUsage, result, err)
				} else {
					splits := regexp.MustCompile("\\s+").Split(result, 6)
					fsUsed := splits[2]
					fsAvail := splits[3]
					fsUsePercent := splits[4]
					fsUsePercentVal, err := strconv.Atoi(strings.Replace(fsUsePercent, "%", "", 1))

					if err != nil {
						integration.PrettyPrintErr("Error determining file system usage percent for %s: %s, %s", fsUsage, result, err)
					} else {
						if fsUsePercentVal < 65 {
							integration.PrettyPrintOk("File system usage of %s amounts to: Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else if fsUsePercentVal < 85 {
							integration.PrettyPrintWarn("File system usage of %s amounts to: Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else {
							integration.PrettyPrintErr("File system usage of %s amounts to: Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						}
					}
				}
			}
		}

		if len(diskSpace.DirectoryUsage) > 0 {
			for _, dirUsage := range diskSpace.DirectoryUsage {
				result, err := integration.PerformSSHCmd(sshOpts, node,
					fmt.Sprintf("du -h -d 0 --exclude=/proc --exclude=/run %s | grep %s", dirUsage, dirUsage),
					clusterStatusOpts.Debug)

				if err != nil {
					integration.PrettyPrintErr("Error estimating directory usage for %s: %s, %s", dirUsage, result, err)
				} else {
					splits := regexp.MustCompile("\\s+").Split(result, 2)
					dirUse := splits[0]

					integration.PrettyPrintOk("Directory usage of %s amounts to %s", dirUsage, dirUse)
				}
			}
		}
	}
}

func checkKubernetesStatus(sshOpts types.SSHConfig, element string,
	resources []types.KubernetesResource, nodes []types.Node) {
	integration.PrintHeader(fmt.Sprintf("Checking status of [%s] ", element), '=')

	if nodes == nil || len(nodes) == 0 {
		integration.PrettyPrintErr("No host configured for [%s]", element)
		os.Exit(1)
	}
	if resources == nil || len(resources) == 0 {
		integration.PrettyPrintErr("No resources configured for [%s]", element)
		os.Exit(1)
	}

	node := integration.GetFirstAccessibleNode(nodes, clusterStatusOpts.Debug)

	if !integration.IsNodeAddressValid(node) {
		integration.PrettyPrintErr("No master available for Kubernetes status check")
		os.Exit(1)
	}

	integration.PrettyPrint("Running kubectl on node %s\n\n", integration.ToNodeLabel(node))

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

		integration.PrettyPrint(msg + namespace_msg + ":\n")
		result, err := integration.PerformSSHCmd(sshOpts, node, command, clusterStatusOpts.Debug)
		integration.PrettyPrint("\n")

		if err != nil {
			integration.PrettyPrintErr("Error checking %s%s: %s, %s", resource.Type, namespace_msg, result, err)
		} else {
			integration.PrettyPrintOk(result)
		}
		integration.PrettyPrint("\n")
	}
}
