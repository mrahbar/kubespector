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
	stopCmd.Flags().StringVarP(&stopOpts.groupArg, "group", "g", "", "Comma-separated list of group names")
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
		integration.PrintHeader(out, fmt.Sprintf("Stopping service %v on node %s (%s):\n",
			stopOpts.targetArg, node.Host, node.IP), '=')
	}
	integration.PrettyPrint(out, "\n")
}

func stopService(sshOpts *integration.SSHConfig, service string, node integration.Node) {
	o, err := integration.PerformSSHCmd(out, sshOpts, &node, fmt.Sprintf("sudo systemctl stop %s", service), RootOpts.Debug)

	integration.PrettyPrint(out, fmt.Sprintf("Result on node %s (%s):\n", node.Host, node.IP))
	if err != nil {
		integration.PrettyPrintErr(out, "Error: %v\nOut: %s", err, strings.TrimSpace(o))
	} else {
		integration.PrettyPrintOk(out, "Service %s stoped. %s", service, strings.TrimSpace(o))
	}

	integration.PrettyPrint(out, "\n")
}
