package cmd

import (
	"fmt"
	"strings"

	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/spf13/cobra"
)

var stopOpts = &CliOpts{}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a Kubernetes service on a target group or node",
	Long: `Service name is mandatory. Either specify node or group in which the service should be stoped.
	When a target group is specified all nodes inside that group will be targeted for service stop.`,
	Run: stopRun,

}

func init() {
	RootCmd.AddCommand(stopCmd)
	stopCmd.Flags().StringVarP(&stopOpts.groupArg, "group", "g", "", "Name of target group")
	stopCmd.Flags().StringVarP(&stopOpts.nodeArg, "node", "n", "", "Name of target node")
	stopCmd.Flags().StringVarP(&stopOpts.targetArg, "service", "s", "", "Name of target service")

}

func stopRun(cmd *cobra.Command, args []string) {
	Run(stopOpts, initializeStopService, stopService)
}

func initializeStopService(service string, node integration.Node, group string) {
	if group != "" {
		integration.PrintHeader(out, fmt.Sprintf("Stopping service %v in group [%s] ",
			service, group), '=')
	} else {
		integration.PrintHeader(out, "Stopping", '=')
	}
	integration.PrettyPrint(out, "\n")
}

func stopService(sshOpts *integration.SSHConfig, service string, node integration.Node) {
	integration.PrettyPrint(out, fmt.Sprintf("Stopping service %v on node %s (%s):\n",
		stopOpts.targetArg, node.Host, node.IP))

	o, err := integration.PerformSSHCmd(out, sshOpts, &node, fmt.Sprintf("sudo systemctl stop %s", service), RootOpts.Debug)

	if err != nil {
		integration.PrettyPrintErr(out, "Error stopping service %s: %v", service, err)
	} else {
		integration.PrettyPrintOk(out, "Service %s stoped. %s", service, strings.TrimSpace(o))
	}

	integration.PrettyPrint(out, "\n")
}
