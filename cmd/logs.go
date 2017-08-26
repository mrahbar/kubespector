package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/spf13/cobra"
)

var logOpts = &types.LogsOpts{}

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:     "logs",
	Short:   "Retrieve logs",
	Long:    `Retrieve logs from system services, containers or pods`,
	PreRunE: util.CheckRequiredFlags,
	Run:     logRun,
}

func init() {
	RootCmd.AddCommand(logsCmd)
	logsCmd.Flags().StringVarP(&logOpts.GroupArg, "group", "g", "", "Comma-separated list of group names")
	logsCmd.Flags().StringVarP(&logOpts.NodeArg, "node", "n", "", "Name of target node")
	logsCmd.Flags().StringVarP(&logOpts.TargetArg, "element", "e", "", "Element to fetch logs from")
	logsCmd.Flags().StringVarP(&logOpts.Type, "type", "t", "", "Element type either service, container or pod")
	logsCmd.Flags().StringVarP(&logOpts.FileOutput, "file", "o", "", "File to save results of command. Screen output is suppressed")
	logsCmd.Flags().StringVarP(&logOpts.Since, "since", "i", "", "Only return logs after a specific timestamp or relative time")
	logsCmd.Flags().IntVarP(&logOpts.Tail, "tail", "a", -1, "Lines of recent log file to display. Defaults to -1 with no selector, showing all log lines")
	logsCmd.Flags().StringArrayVarP(&logOpts.ExtraArgs, "extra-arg", "r", []string{}, "Additional command line args to execute")
	logsCmd.Flags().BoolVarP(&logOpts.Sudo, "sudo", "s", false, "Run as sudo")

	logsCmd.MarkFlagRequired("element")
	logsCmd.MarkFlagRequired("type")
}

func logRun(_ *cobra.Command, _ []string) {
	config := util.UnmarshalConfig()
    params := &types.CommandParams{
        Printer: printer,
        Config:  config,
        Opts:    logOpts,
    }
    pkg.Logs(params)
}
