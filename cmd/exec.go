package cmd

import (
	"fmt"
	"strings"

	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/spf13/cobra"
)

type execCliOpts struct {
	groupArg  string
	nodeArg   string
	targetArg string
	sudo      bool
}

var execOpts = &execCliOpts{}

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Executes a command on a target group or node",
	Long: `Command to execute is mandatory. Either specify node or group on which command should be executed.
	When a target group is specified all nodes inside that group will be targeted.`,
	PreRunE: util.CheckRequiredFlags,
	Run:     execRun,
}

func init() {
	RootCmd.AddCommand(execCmd)
	execCmd.Flags().StringVarP(&execOpts.groupArg, "group", "g", "", "Comma-separated list of group names")
	execCmd.Flags().StringVarP(&execOpts.nodeArg, "node", "n", "", "Name of target node")
	execCmd.Flags().StringVarP(&execOpts.targetArg, "cmd", "c", "", "Command to execute")
	execCmd.Flags().BoolVarP(&execOpts.sudo, "sudo", "s", false, "Run as sudo")

	execCmd.MarkFlagRequired("cmd")
}

func execRun(_ *cobra.Command, _ []string) {
	opts := &CliOpts{
		groupArg:  execOpts.groupArg,
		nodeArg:   execOpts.nodeArg,
		targetArg: execOpts.targetArg,
	}

	Run(opts, initializeExec, exec)
}

func initializeExec(target string, node string, group string) {
	if group != "" {
		integration.PrintHeader(out, fmt.Sprintf("Executing '%v' in group [%s] ",
			target, group), '=')
	}

	if node != "" {
		integration.PrintHeader(out, fmt.Sprintf("Executing '%v' on node %s :\n",
			execOpts.targetArg, node), '=')
	}

	integration.PrettyPrint(out, "\n")
}

func exec(sshOpts *integration.SSHConfig, command string, node integration.Node) {
	command = fmt.Sprintf("bash -c '%s'", command)

	o, err := integration.PerformSSHCmd(out, sshOpts, &node, command, RootOpts.Debug)

	integration.PrettyPrint(out, fmt.Sprintf("Result on node %s:\n", util.ToNodeLabel(node)))
	if err != nil {
		integration.PrettyPrintErr(out, "Error: %v\nOut: %s", err, strings.TrimSpace(o))
	} else {
		integration.PrettyPrintOk(out, "%s", strings.TrimSpace(o))
	}

	integration.PrettyPrint(out, "\n")
}
