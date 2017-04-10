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
)

type statusCliOpts struct {
	groupsArg string
	nodeArg   string
}

var groups = ClusterMembers
var statusOpts = &statusCliOpts{}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of Kubernetes nodes (services, disk space) defined in configuration file",
	Long:  `When called without arguments all hosts in configuration will be examined.`,
	Run:   statusRun,
}

func init() {
	RootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVarP(&statusOpts.groupsArg, "groups", "g", "", "Comma-separated list of group names")
	statusCmd.Flags().StringVarP(&statusOpts.nodeArg, "node", "n", "", "Name of target node")
}

func statusRun(cmd *cobra.Command, args []string) {
	var config integration.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		integration.PrettyPrintErr(out, "Unable to decode config: %v", err)
		os.Exit(1)
	} else {
		if statusOpts.groupsArg != "" {
			groups = strings.Split(statusOpts.groupsArg, ",")
			fmt.Printf("Restricted status check to groups: %v\n", strings.Join(groups, " "))
		} else {
			fmt.Printf("Performing status check for groups: %v\n", strings.Join(groups, " "))
		}

		for _, element := range groups {
			switch element {
			case "Etcd":
				checkServiceStatus(&config.Ssh, element, config.Cluster.Etcd.Services, config.Cluster.Etcd.Nodes)
				checkDiskStatus(&config.Ssh, element, config.Cluster.Etcd.DiskSpace, config.Cluster.Etcd.Nodes)
			case "Master":
				checkServiceStatus(&config.Ssh, element, config.Cluster.Master.Services, config.Cluster.Master.Nodes)
				checkDiskStatus(&config.Ssh, element, config.Cluster.Master.DiskSpace, config.Cluster.Master.Nodes)
			case "Worker":
				checkServiceStatus(&config.Ssh, element, config.Cluster.Worker.Services, config.Cluster.Worker.Nodes)
				checkDiskStatus(&config.Ssh, element, config.Cluster.Worker.DiskSpace, config.Cluster.Worker.Nodes)
			case "Ingress":
				checkServiceStatus(&config.Ssh, element, config.Cluster.Ingress.Services, config.Cluster.Ingress.Nodes)
				checkDiskStatus(&config.Ssh, element, config.Cluster.Ingress.DiskSpace, config.Cluster.Ingress.Nodes)
			case "Registry":
				checkServiceStatus(&config.Ssh, element, config.Cluster.Registry.Services, config.Cluster.Registry.Nodes)
				checkDiskStatus(&config.Ssh, element, config.Cluster.Registry.DiskSpace, config.Cluster.Registry.Nodes)
			case "Kubernetes":
				checkKubernetesStatus(&config.Ssh, element, config.Kubernetes.Resources, config.Cluster.Master.Nodes)
			}
		}
	}
}

func checkServiceStatus(sshOpts *integration.SSHConfig, element string, services []string, nodes []integration.Node) {
	integration.PrintHeader(out, fmt.Sprintf("Checking service status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0  {
		integration.PrettyPrintIgnored(out, "No host configured for [%s]", element)
		return
	}
	if services == nil || len(services) == 0   {
		integration.PrettyPrintIgnored(out, "No services configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		if !util.IsNodeAddressValid(node) {
			integration.PrettyPrintErr(out, "Current node %q has no valid address", node)
			break
		}

		integration.PrettyPrint(out, "\nOn node %s (%s):\n", node.Host, node.IP)

		for _, service := range services {
			o, err := integration.PerformSSHCmd(out, sshOpts, &node,
				fmt.Sprintf("systemctl is-active %s", service), RootOpts.Debug)
			result := strings.TrimSpace(o)

			if err != nil {
				integration.PrettyPrintErr(out, "Error checking status of %s: %s, %s", service, result, err)
			} else {

				if result == "active" {
					integration.PrettyPrintOk(out, "Service %s is active", service)
				} else if result == "activating" {
					integration.PrettyPrintWarn(out, "Service %s is activating", service)
				} else if result == "inactive" {
					integration.PrettyPrintWarn(out, "Service %s is inactive", service)
				} else if result == "failed" {
					integration.PrettyPrintErr(out, "Service %s is failed", service)
				} else {
					integration.PrettyPrintUnknown(out, "Service %s is unknown state", service)
				}
			}
		}
	}
}

func checkDiskStatus(sshOpts *integration.SSHConfig, element string, diskSpace integration.DiskSpace, nodes []integration.Node) {
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

		integration.PrettyPrint(out, "\nOn node %s (%s):\n", node.Host, node.IP)


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
							integration.PrettyPrintOk(out, "File system usage of %s amounts to:\n Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else if fsUsePercentVal < 85 {
							integration.PrettyPrintWarn(out, "File system usage of %s amounts to:\n Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else {
							integration.PrettyPrintErr(out, "File system usage of %s amounts to:\n Used: %s Available: %s (%s)",
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
	resources []integration.KubernetesResource, nodes []integration.Node)  {
	integration.PrintHeader(out, fmt.Sprintf("Checking status of [%s] ", element), '=')

	if nodes == nil || len(nodes) == 0  {
		integration.PrettyPrintErr(out, "No master host configured for [%s]", element)
		os.Exit(1)
	}
	if resources == nil || len(resources) == 0   {
		integration.PrettyPrintErr(out, "No resources configured for [%s]", element)
		os.Exit(1)
	}

	node := nodes[0]
	integration.PrettyPrint(out, "Running kubectl on node %s (%s)\n", node.Host, node.IP)

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

		integration.PrettyPrint(out, msg+namespace_msg+"\n")
		o, err := integration.PerformSSHCmd(out, sshOpts, &node, command, RootOpts.Debug)
		result := strings.TrimSpace(o)
		integration.PrettyPrint(out, "\n")

		if err != nil {
			integration.PrettyPrintErr(out, "Error checking %s%s: %s, %s", resource.Type, namespace_msg, result, err)
		} else {
			integration.PrettyPrintOk(out, result)
		}
	}
}