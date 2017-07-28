package cmd

import (
	"fmt"
	"strings"

	"github.com/mrahbar/kubernetes-inspector/integration"

	"github.com/spf13/cobra"
)

var statusOpts = &CliOpts{}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of a service on a target group or node",
	Long: `Service name is mandatory. Either specify node or group in which the service status should be checked.
	When a target group is specified all nodes inside that group will be targeted.`,
	PreRunE: integration.CheckRequiredFlags,
	Run:     statusRun,
}

func init() {
	ServiceCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVarP(&statusOpts.groupArg, "group", "g", "", "Comma-separated list of group names")
	statusCmd.Flags().StringVarP(&statusOpts.nodeArg, "node", "n", "", "Name of target node")
	statusCmd.Flags().StringVarP(&statusOpts.targetArg, "service", "s", "", "Name of target service")
	statusCmd.MarkFlagRequired("service")
}

func statusRun(_ *cobra.Command, _ []string) {
	Run(statusOpts, initializeStatusService, statusService)
}

func initializeStatusService(service string, node string, group string) {
	if group != "" {
		integration.PrintHeader(out, fmt.Sprintf("Checking status of service %v in group [%s] ",
			service, group), '=')
	}

	if node != "" {
		integration.PrintHeader(out, fmt.Sprintf("Checking status of service %v on node %s:\n",
			statusOpts.targetArg, node), '=')
	}

	integration.PrettyPrint(out, "\n")
}

func statusService(sshOpts integration.SSHConfig, service string, node integration.Node) {
	o, err := integration.PerformSSHCmd(out, sshOpts, node, fmt.Sprintf("sudo systemctl status %s -l", service), RootOpts.Debug)

	integration.PrettyPrint(out, fmt.Sprintf("Result on node %s:\n", integration.ToNodeLabel(node)))
	if err != nil {
		integration.PrettyPrintErr(out, "Error: %v\nOut: %s", err, strings.TrimSpace(o))
	} else {
		integration.PrettyPrintOk(out, "%s", strings.TrimSpace(o))
	}

	integration.PrettyPrint(out, "\n")
}
