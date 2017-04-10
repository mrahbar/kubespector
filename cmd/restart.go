package cmd

import (
	"fmt"
	"strings"

	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/spf13/cobra"
)

var restartOpts = &CliOpts{}

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
	restartCmd.Flags().StringVarP(&restartOpts.targetArg, "service", "s", "", "Name of target service")

}

func restartRun(cmd *cobra.Command, args []string) {
	Run(restartOpts, initializeRestartService, restartService)
}

func initializeRestartService(service string, node integration.Node, group string) {
	if group != "" {
		integration.PrintHeader(out, fmt.Sprintf("Restarting service %v in group [%s] ",
			service, group), '=')
	} else {
		integration.PrintHeader(out, "Restarting", '=')
	}
}

func restartService(sshOpts *integration.SSHConfig, service string, node integration.Node) {
	integration.PrettyPrint(out, fmt.Sprintf("Restarting service %v on node %s (%s):\n",
		restartOpts.targetArg, node.Host, node.IP))

	o, err := integration.PerformSSHCmd(out, sshOpts, &node, fmt.Sprintf("sudo systemctl restart %s", service), RootOpts.Debug)

	if err != nil {
		integration.PrettyPrintErr(out, "Error restarting service %s: %v", service, err)
	} else {
		integration.PrettyPrintOk(out, "Service %s restarted. %s", service, strings.TrimSpace(o))
	}

	integration.PrettyPrint(out, "\n")
}
