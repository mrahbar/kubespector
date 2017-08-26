package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/util"

	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/spf13/cobra"
)

var execOpts = &types.ExecOpts{}

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
	execCmd.Flags().StringVarP(&execOpts.GroupArg, "group", "g", "", "Comma-separated list of group names")
	execCmd.Flags().StringVarP(&execOpts.NodeArg, "node", "n", "", "Name of target node")
	execCmd.Flags().StringVarP(&execOpts.TargetArg, "cmd", "c", "", "Command to execute")
	execCmd.Flags().StringVarP(&execOpts.FileOutput, "file", "o", "", "File to save results of command. Screen output is suppressed")
	execCmd.Flags().BoolVarP(&execOpts.Sudo, "sudo", "s", false, "Run as sudo")

	execCmd.MarkFlagRequired("cmd")
}

func execRun(_ *cobra.Command, _ []string) {
	config := util.UnmarshalConfig()
    params := &types.CommandParams{
        Printer: printer,
        Config:  config,
        Opts:    execOpts,
    }
    pkg.Exec(params)
}
