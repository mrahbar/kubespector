package cmd

import (
	"fmt"
	"strings"

	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/spf13/cobra"
)

var execOpts = &CliOpts{}

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Executes a command on a target group or node",
	Long: `Command to execute is mandatory. Either specify node or group on which command should be executed.
	When a target group is specified all nodes inside that group will be targeted.`,
	Run: execRun,

}

func init() {
	RootCmd.AddCommand(execCmd)
	execCmd.Flags().StringVarP(&execOpts.groupArg, "group", "g", "", "Name of target group")
	execCmd.Flags().StringVarP(&execOpts.nodeArg, "node", "n", "", "Name of target node")
	execCmd.Flags().StringVarP(&execOpts.targetArg, "exec", "e", "", "Command to execute")

}

func execRun(cmd *cobra.Command, args []string) {
	Run(execOpts, initializeExec, exec)
}

func initializeExec(target string, node integration.Node, group string) {
	if group != "" {
		integration.PrintHeader(out, fmt.Sprintf("Executing '%v' in group [%s] ",
			target, group), '=')
	} else {
		integration.PrintHeader(out, "Executing", '=')
	}
}

func exec(sshOpts *integration.SSHConfig, command string, node integration.Node) {
	integration.PrettyPrint(out, fmt.Sprintf("Executing '%v' on node %s (%s):\n",
		execOpts.targetArg, node.Host, node.IP))

	o, err := integration.PerformSSHCmd(out, sshOpts, &node, fmt.Sprintf("sudo %s", command), RootOpts.Debug)

	if err != nil {
		integration.PrettyPrintErr(out, "Error executing command '%v': %v", command, err)
	} else {
		integration.PrettyPrintOk(out, "Command '%v' executed. %s", command, strings.TrimSpace(o))
	}

	integration.PrettyPrint(out, "\n")
}
