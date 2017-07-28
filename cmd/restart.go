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
	PreRunE: integration.CheckRequiredFlags,
	Run:     restartRun,
}

func init() {
	ServiceCmd.AddCommand(restartCmd)
	restartCmd.Flags().StringVarP(&restartOpts.groupArg, "group", "g", "", "Comma-separated list of group names")
	restartCmd.Flags().StringVarP(&restartOpts.nodeArg, "node", "n", "", "Name of target node")
	restartCmd.Flags().StringVarP(&restartOpts.targetArg, "service", "s", "", "Name of target service")
	restartCmd.MarkFlagRequired("service")
}

func restartRun(_ *cobra.Command, _ []string) {
	Run(restartOpts, initializeRestartService, restartService)
}

func initializeRestartService(service string, node string, group string) {
	if group != "" {
		integration.PrintHeader(out, fmt.Sprintf("Restarting service %v in group [%s] ",
			service, group), '=')
	}

	if node != "" {
		integration.PrintHeader(out, fmt.Sprintf("Restarting service %v on node %s:\n",
			restartOpts.targetArg, node), '=')
	}

	integration.PrettyPrint(out, "\n")
}

func restartService(sshOpts integration.SSHConfig, service string, node integration.Node) {
	o, err := integration.PerformSSHCmd(out, sshOpts, node, fmt.Sprintf("sudo systemctl restart %s", service), RootOpts.Debug)

	integration.PrettyPrint(out, fmt.Sprintf("Result on node %s:\n", integration.ToNodeLabel(node)))

	if err != nil {
		integration.PrettyPrintErr(out, "Error: %v\nOut: %s", err, strings.TrimSpace(o))
	} else {
		integration.PrettyPrintOk(out, "Service %s restarted. %s", service, strings.TrimSpace(o))
	}

	integration.PrettyPrint(out, "\n")
}
