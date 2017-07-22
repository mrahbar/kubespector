package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"text/template"
	"bytes"
)

const (
	leftTemplateDelim  = "{{"
	rightTemplateDelim = "}}"
)

var clusterStatusChecks = []string{integration.SERVICES_CHECKNAME, integration.CONTAINERS_CHECKNAME, integration.CERTIFICATES_CHECKNAME, integration.DISKUSAGE_CHECKNAME}

type clusterStatusCliOpts struct {
	groupsArg string
	checksArg string
}

var clusterStatusOpts = &clusterStatusCliOpts{}

// clusterStatusCmd represents the clusterStatus command
var clusterStatusCmd = &cobra.Command{
	Use:     "cluster-status",
	Aliases: []string{"cs"},
	Short:   "Performs various checks on the cluster defined in the configuration file",
	Long:    `When called without arguments all hosts and checks in configuration will be executed.`,
	Run:     clusterStatusRun,
}

func init() {
	RootCmd.AddCommand(clusterStatusCmd)
	clusterStatusCmd.Flags().StringVarP(&clusterStatusOpts.groupsArg, "groups", "g", "", "Comma-separated list of group names")
	clusterStatusCmd.Flags().StringVarP(&clusterStatusOpts.checksArg, "checks", "c", "", "Comma-separated list of checks. E.g. Services,Containers")
}

func clusterStatusRun(_ *cobra.Command, _ []string) {
	var config integration.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		integration.PrettyPrintErr(out, "Unable to decode config: %v", err)
		os.Exit(1)
	} else {
		var groups = []string{}
		if clusterStatusOpts.groupsArg != "" {
			groups = strings.Split(clusterStatusOpts.groupsArg, ",")
		} else {
			for _, group := range config.ClusterGroups {
				groups = append(groups, group.Name)
			}
			groups = append(groups, integration.KUBERNETES_GROUPNAME)
		}

		if clusterStatusOpts.checksArg != "" {
			clusterStatusChecks = strings.Split(clusterStatusOpts.checksArg, ",")
		}

		integration.PrettyPrint(out, "Performing status checks %s for groups: %v\n", strings.Join(groups, ","), strings.Join(groups, " "))

		for _, element := range groups {
			if element != integration.KUBERNETES_GROUPNAME {
				group := util.FindGroupByName(config.ClusterGroups, element)
				if group.Nodes != nil {
					if util.ElementInArray(clusterStatusChecks, integration.SERVICES_CHECKNAME) {
						checkServiceStatus(&config.Ssh, element, group.Services, group.Nodes)
					}

					if util.ElementInArray(clusterStatusChecks, integration.CONTAINERS_CHECKNAME) {
						checkContainerStatus(&config.Ssh, element, group.Containers, group.Nodes)
					}

					if util.ElementInArray(clusterStatusChecks, integration.CERTIFICATES_CHECKNAME) {
						checkCertificatesExpiration(&config.Ssh, element, group.Certificates, group.Nodes)
					}

					if util.ElementInArray(clusterStatusChecks, integration.DISKUSAGE_CHECKNAME) {
						checkDiskStatus(&config.Ssh, element, group.DiskUsage, group.Nodes)
					}
				} else {
					integration.PrettyPrintErr(out, "No Nodes found for group: %s", element)
				}
			} else {
				group := util.FindGroupByName(config.ClusterGroups, integration.MASTER_GROUPNAME)
				checkKubernetesStatus(&config.Ssh, element, config.Kubernetes.Resources, group.Nodes)
			}
		}
	}
}

func checkServiceStatus(sshOpts *integration.SSHConfig, element string, services []string, nodes []integration.Node) {
	integration.PrintHeader(out, fmt.Sprintf("Checking service status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		integration.PrettyPrintIgnored(out, "No host configured for [%s]", element)
		return
	}
	if services == nil || len(services) == 0 {
		integration.PrettyPrintIgnored(out, "No services configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			integration.PrettyPrintErr(out, "Current node %q has no valid address", node)
			break
		}

		integration.PrettyPrint(out, "\nOn node %s:\n", util.ToNodeLabel(node))

		for _, service := range services {
			o, err := integration.PerformSSHCmd(out, sshOpts, &node,
				fmt.Sprintf("systemctl is-active %s", service), RootOpts.Debug)
			result := strings.TrimSpace(o)

			if err != nil {
				integration.PrettyPrintErr(out, "Error checking status of %s: %s, %s", service, result, err)
			} else {
				if result == "active" {
					integration.PrettyPrintOk(out, "Service %s is active", service)
				} else if result == "activating" || result == "inactive" {
					integration.PrettyPrintWarn(out, "Service %s is %s", service, result)
				} else if result == "failed" {
					integration.PrettyPrintErr(out, "Service %s is failed", service)
				} else {
					integration.PrettyPrintUnknown(out, "Service %s is unknown state: %s", service, restartCmd)
				}
			}
		}
	}
}

func checkContainerStatus(sshOpts *integration.SSHConfig, element string, containers []string, nodes []integration.Node) {
	integration.PrintHeader(out, fmt.Sprintf("Checking container status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		integration.PrettyPrintIgnored(out, "No host configured for [%s]", element)
		return
	}
	if containers == nil || len(containers) == 0 {
		integration.PrettyPrintIgnored(out, "No containers configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			integration.PrettyPrintErr(out, "Current node %q has no valid address", node)
			break
		}

		integration.PrettyPrint(out, "\nOn node %s:\n", util.ToNodeLabel(node))

		for _, container := range containers {
			o, err := integration.PerformSSHCmd(out, sshOpts, &node,
				fmt.Sprintf("bash -c 'docker ps -a --latest -f name=%s* -q | xargs --no-run-if-empty docker inspect -f '{{.State.Status}}''", container), RootOpts.Debug)
			result := strings.TrimSpace(o)

			if err != nil {
				integration.PrettyPrintErr(out, "Error checking status of %s: %s, %s", container, result, err)
			} else {
				if result == "running" {
					integration.PrettyPrintOk(out, "Container %s is running", container)
				} else if result == "created" || result == "paused" || result == "restarting" {
					integration.PrettyPrintWarn(out, "Container %s is %s", container, result)
				} else if result == "exited" || result == "removing" || result == "dead" {
					integration.PrettyPrintErr(out, "Container %s is %s", container, result)
				} else {
					integration.PrettyPrintUnknown(out, "Container %s not found or in unknown state: %s", container, result)
				}
			}
		}
	}
}

func checkCertificatesExpiration(sshOpts *integration.SSHConfig, element string, certificates []string, nodes []integration.Node) {
	integration.PrintHeader(out, fmt.Sprintf("Checking certificate status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		integration.PrettyPrintIgnored(out, "No host configured for [%s]", element)
		return
	}
	if certificates == nil || len(certificates) == 0 {
		integration.PrettyPrintIgnored(out, "No certificates configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			integration.PrettyPrintErr(out, "Current node %q has no valid address", node)
			break
		}

		integration.PrettyPrint(out, "\nOn node %s:\n", util.ToNodeLabel(node))

		for _, cert := range certificates {
			cert = parseTemplate(cert, node, RootOpts.Debug)
			o, err := integration.PerformSSHCmd(out, sshOpts, &node,
				fmt.Sprintf("bash -c 'openssl x509 -enddate -noout -in %s |cut -d= -f 2'", cert), RootOpts.Debug)
			result := strings.TrimSpace(o)

			if err != nil {
				integration.PrettyPrintErr(out, "Error checking expiration of %s: %s, %s", cert, result, err)
			} else {
				_, err = integration.PerformSSHCmd(out, sshOpts, &node,
					fmt.Sprintf("openssl x509 -checkend 86400 -noout -in %s", cert), RootOpts.Debug)

				if err == nil {
					integration.PrettyPrintOk(out, "Certificate %s is valid until %s", cert, result)
				} else {
					integration.PrettyPrintWarn(out, "Certificate %s is only valid until %s", cert, result)
				}
			}
		}
	}
}

func parseTemplate(value string, node integration.Node, debug bool) string {
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

func checkDiskStatus(sshOpts *integration.SSHConfig, element string, diskSpace integration.DiskUsage, nodes []integration.Node) {
	integration.PrintHeader(out, fmt.Sprintf("Checking disk status of group [%s] ", element), '-')
	if nodes == nil || len(nodes) == 0 {
		integration.PrettyPrintIgnored(out, "No host configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			integration.PrettyPrintErr(out, "Current node %q has no valid address", node)
			break
		}

		integration.PrettyPrint(out, "\nOn node %s:\n", util.ToNodeLabel(node))

		if len(diskSpace.FileSystemUsage) > 0 {
			for _, fsUsage := range diskSpace.FileSystemUsage {
				o, err := integration.PerformSSHCmd(out, sshOpts, &node,
					fmt.Sprintf("df -h | grep %s", fsUsage), RootOpts.Debug)
				result := strings.TrimSpace(o)

				if err != nil {
					integration.PrettyPrintErr(out, "Error estimating file system usage for %s: %s, %s", fsUsage, result, err)
				} else {
					splits := regexp.MustCompile("\\s+").Split(result, 6)
					fsUsed := splits[2]
					fsAvail := splits[3]
					fsUsePercent := splits[4]
					fsUsePercentVal, err := strconv.Atoi(strings.Replace(fsUsePercent, "%", "", 1))

					if err != nil {
						integration.PrettyPrintErr(out, "Error determining file system usage percent for %s: %s, %s", fsUsage, o, err)
					} else {
						if fsUsePercentVal < 65 {
							integration.PrettyPrintOk(out, "File system usage of %s amounts to: Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else if fsUsePercentVal < 85 {
							integration.PrettyPrintWarn(out, "File system usage of %s amounts to: Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else {
							integration.PrettyPrintErr(out, "File system usage of %s amounts to: Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						}
					}
				}
			}
		}

		if len(diskSpace.DirectoryUsage) > 0 {
			for _, dirUsage := range diskSpace.DirectoryUsage {
				o, err := integration.PerformSSHCmd(out, sshOpts, &node,
					fmt.Sprintf("du -h -d 0 --exclude=/proc --exclude=/run %s | grep %s", dirUsage, dirUsage),
					RootOpts.Debug)
				result := strings.TrimSpace(o)

				if err != nil {
					integration.PrettyPrintErr(out, "Error estimating directory usage for %s: %s, %s", dirUsage, result, err)
				} else {
					splits := regexp.MustCompile("\\s+").Split(result, 2)
					dirUse := splits[0]

					integration.PrettyPrintOk(out, "Directory usage of %s amounts to %s", dirUsage, dirUse)
				}
			}
		}
	}
}

func checkKubernetesStatus(sshOpts *integration.SSHConfig, element string,
	resources []integration.KubernetesResource, nodes []integration.Node) {
	integration.PrintHeader(out, fmt.Sprintf("Checking status of [%s] ", element), '=')

	if nodes == nil || len(nodes) == 0 {
		integration.PrettyPrintErr(out, "No master host configured for [%s]", element)
		os.Exit(1)
	}
	if resources == nil || len(resources) == 0 {
		integration.PrettyPrintErr(out, "No resources configured for [%s]", element)
		os.Exit(1)
	}

	node := util.RetrieveKubectlNode(nodes, RootOpts.Debug)

	if !util.IsNodeAddressValid(node) {
		integration.PrettyPrintErr(out, "No master available for Kubernetes status check")
		os.Exit(1)
	}

	integration.PrettyPrint(out, "Running kubectl on node %s\n\n", util.ToNodeLabel(node))

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

		integration.PrettyPrint(out, msg+namespace_msg+":\n")
		o, err := integration.PerformSSHCmd(out, sshOpts, &node, command, RootOpts.Debug)
		result := strings.TrimSpace(o)
		integration.PrettyPrint(out, "\n")

		if err != nil {
			integration.PrettyPrintErr(out, "Error checking %s%s: %s, %s", resource.Type, namespace_msg, result, err)
		} else {
			integration.PrettyPrintOk(out, result)
		}
		integration.PrettyPrint(out, "\n")
	}
}
