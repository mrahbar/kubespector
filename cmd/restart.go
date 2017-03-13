package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"os"
	"reflect"
	"strings"
)

type restartCliOpts struct {
	groupArg   string
	nodeArg    string
	serviceArg string
}

var restartOpts = &restartCliOpts{}

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restarts a Kubernetes service on a target group or node",
	Long: `Service name is mandatory. Either specify node or group in which the service should be restarted.
	When a target group is specified all nodes inside that group will be targeted for service restart.`,
	Run:   restartRun,
}

func init() {
	RootCmd.AddCommand(restartCmd)
	restartCmd.Flags().StringVarP(&restartOpts.groupArg, "group", "g", "", "Name of target group")
	restartCmd.Flags().StringVarP(&restartOpts.nodeArg, "node", "n", "", "Name of target node")
	restartCmd.Flags().StringVarP(&restartOpts.serviceArg, "service", "s", "", "Name of target service")

}

func restartRun(cmd *cobra.Command, args []string) {
	var config integration.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		integration.PrettyPrintErr(out, "Unable to decode config: %v", err)
		os.Exit(1)
	} else {
		if restartOpts.serviceArg == "" {
			integration.PrettyPrintErr(out, "Command restart has to be called with a service name.")
			os.Exit(1)
		}

		if restartOpts.nodeArg != "" {
			v := reflect.ValueOf(config.Cluster)
			node := integration.Node{}

			for i := 0; i < v.NumField(); i++ {
				nodes := v.Field(i).FieldByName("Nodes").Interface().([]integration.Node)
				for _, n := range nodes {
					if n.Host == restartOpts.nodeArg || n.IP == restartOpts.nodeArg {
						node = n
						break
					}
				}
			}

			if node.IP != "" {
				restartService(&config.Ssh, restartOpts.serviceArg, node)
			} else {
				integration.PrettyPrintErr(out, "No node found for %v in config", restartOpts.nodeArg)
				os.Exit(1)
			}

		} else {
			if restartOpts.groupArg == "" {
				integration.PrettyPrintErr(out, "Command restart has to be called with a group name")
				os.Exit(1)
			}

			var nodes []integration.Node

			switch restartOpts.groupArg {
			case "Etcd":
				nodes = config.Cluster.Etcd.Nodes
			case "Master":
				nodes = config.Cluster.Master.Nodes
			case "Worker":
				nodes = config.Cluster.Worker.Nodes
			case "Ingress":
				nodes = config.Cluster.Ingress.Nodes
			case "Registry":
				nodes = config.Cluster.Registry.Nodes
			}

			if nodes == nil {
				integration.PrettyPrintErr(out, "Group name is not in list of available groups: %s", ClusterMembers)
				os.Exit(1)
			}

			integration.PrintHeader(out, fmt.Sprintf("Restarting service %v in group [%s] ",
				restartOpts.serviceArg, restartOpts.groupArg), '=')
			for _, node := range nodes {
				restartService(&config.Ssh, restartOpts.serviceArg, node)
			}
		}
	}
}

func restartService(sshOpts *integration.SSHConfig, service string, node integration.Node) {
	host_msg := " "
	ip_msg := " "
	if node.Host != "" {
		host_msg += node.Host
	}
	if node.IP == "" {
		integration.PrettyPrintErr(out, "Current node%s has no IP configured", host_msg)
		return
	}

	ip_msg += "(" + node.IP + "):\n"
	integration.PrettyPrint(out, fmt.Sprintf("Restarting service %v on node%s%s",
		restartOpts.serviceArg, host_msg, ip_msg))

	o, err := integration.PerformSSHCmd(out, sshOpts, &node, fmt.Sprintf("sudo systemctl restart %s", service), RootOpts.Debug)

	if err != nil {
		integration.PrettyPrintErr(out, "Error checking status of %s: %v", service, err)
	} else {
		integration.PrettyPrintOk(out, "Service %s restarted. %s", service, strings.TrimSpace(o))
	}
	integration.PrettyPrint(out, "\n")
}
