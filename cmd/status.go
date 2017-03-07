package cmd

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/spf13/cobra"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/util"
	"strings"
	"regexp"
	"strconv"
	"os"
)

type statusCliOpts struct {
	groupsArg string
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
}

func statusRun(cmd *cobra.Command, args []string) {
	var config integration.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		util.PrettyPrintErr(out, "Unable to decode config: %v", err)
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
	util.PrintHeader(out, fmt.Sprintf("Checking service status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0  {
		util.PrettyPrintIgnored(out, "No host configured for [%s]", element)
		return
	}
	if services == nil || len(services) == 0   {
		util.PrettyPrintIgnored(out, "No services configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		host_msg := " "
		ip_msg := " "
		if node.Host != "" {
			host_msg += node.Host
		}
		if node.IP == "" {
			util.PrettyPrintWarn(out, "Current node%s has no IP configured", host_msg)
			break
		}

		ip_msg += "(" + node.IP + "):\n"
		util.PrettyPrint(out, "On host %s%s", host_msg, ip_msg)

		for _, service := range services {
			o, err := integration.PerformSSHCmd(out, sshOpts, &node,
				fmt.Sprintf("sudo systemctl is-active %s", service), RootOpts.Debug)
			result := strings.TrimSpace(o)

			if err != nil {
				util.PrettyPrintErr(out, "Error checking status of %s: %s, %s", service, result, err)
			} else {

				if result == "active" {
					util.PrettyPrintOk(out, "Service %s is active", service)
				} else if result == "activating" {
					util.PrettyPrintWarn(out, "Service %s is activating", service)
				} else if result == "inactive" {
					util.PrettyPrintWarn(out, "Service %s is inactive", service)
				} else if result == "failed" {
					util.PrettyPrintErr(out, "Service %s is failed", service)
				} else {
					util.PrettyPrintUnknown(out, "Service %s is unknown state", service)
				}
			}
		}
	}
}

func checkDiskStatus(sshOpts *integration.SSHConfig, element string, diskSpace integration.DiskSpace, nodes []integration.Node) {
	util.PrintHeader(out, fmt.Sprintf("Checking disk status of group [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0 {
		util.PrettyPrintIgnored(out, "No host configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		host_msg := " "
		ip_msg := " "
		if node.Host != "" {
			host_msg += node.Host
		}
		if node.IP == "" {
			util.PrettyPrintWarn(out, "Current node%s has no IP configured", host_msg)
			break
		}

		ip_msg += "(" + node.IP + "):\n"
		util.PrettyPrint(out, "On host%s%s", host_msg, ip_msg)

		if len(diskSpace.FileSystemUsage) > 0 {
			for _, fsUsage := range diskSpace.FileSystemUsage {
				o, err := integration.PerformSSHCmd(out, sshOpts, &node,
					fmt.Sprintf("df -h | grep %s", fsUsage), RootOpts.Debug)
				result := strings.TrimSpace(o)

				if err != nil {
					util.PrettyPrintErr(out, "Error estimating file system usage for %s: %s, %s", fsUsage, result, err)
				} else {
					splits := regexp.MustCompile("\\s+").Split(result, 6)
					fsUsed := splits[2]
					fsAvail := splits[3]
					fsUsePercent := splits[4]
					fsUsePercentVal, err := strconv.Atoi(strings.Replace(fsUsePercent, "%", "", 1))

					if err != nil {
						util.PrettyPrintErr(out, "Error determining file system usage percent for %s: %s, %s", fsUsage, o, err)
					} else {
						if fsUsePercentVal < 65 {
							util.PrettyPrintOk(out, "File system usage of %s amounts to Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else if fsUsePercentVal < 85 {
							util.PrettyPrintWarn(out, "File system usage of %s amounts to Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						} else {
							util.PrettyPrintErr(out, "File system usage of %s amounts to Used: %s Available: %s (%s)",
								fsUsage, fsUsed, fsAvail, fsUsePercent)
						}
					}
				}
			}
		}

		if len(diskSpace.DirectoryUsage) > 0 {
			for _, dirUsage := range diskSpace.DirectoryUsage {
				o, err := integration.PerformSSHCmd(out, sshOpts, &node,
					fmt.Sprintf("sudo du -h -d 0 --exclude=/proc --exclude=/run %s | grep %s", dirUsage, dirUsage),
					RootOpts.Debug)
				result := strings.TrimSpace(o)

				if err != nil {
					util.PrettyPrintErr(out, "Error estimating directory usage for %s: %s, %s", dirUsage, result, err)
				} else {
					splits := regexp.MustCompile("\\s+").Split(result, 2)
					dirUse := splits[0]

					util.PrettyPrintOk(out, "Directory usage of %s amounts to %s", dirUsage, dirUse)
				}
			}
		}
	}
}

func checkKubernetesStatus(sshOpts *integration.SSHConfig, element string,
	resources []integration.KubernetesResource, nodes []integration.Node)  {
	util.PrintHeader(out, fmt.Sprintf("Checking status of [%s] ", element), '=')

	if nodes == nil || len(nodes) == 0  {
		util.PrettyPrintErr(out, "No master host configured for [%s]", element)
		os.Exit(1)
	}
	if resources == nil || len(resources) == 0   {
		util.PrettyPrintErr(out, "No resources configured for [%s]", element)
		os.Exit(1)
	}

	node := nodes[0]
	util.PrettyPrint(out, "Running kubectl on node %s (%s)\n", node.Host, node.IP)

	for _, resource := range resources {
		msg := fmt.Sprintf("Status of %s", resource.Type)
		namespace_msg := ""
		command := fmt.Sprintf("sudo kubectl get %s", resource.Type)
		if resource.Namespace != "" {
			namespace_msg += " in namespace " + resource.Namespace
			command += " -n " + resource.Namespace
		}
		if resource.Wide {
			command += " -o wide"
		}

		util.PrettyPrint(out, msg+namespace_msg+"\n")
		o, err := integration.PerformSSHCmd(out, sshOpts, &node, command, RootOpts.Debug)
		result := strings.TrimSpace(o)

		if err != nil {
			util.PrettyPrintErr(out, "Error checking %s%s: %s, %s", resource.Type, namespace_msg, result, err)
		} else {
			util.PrettyPrintOk(out, result)
		}
	}
}