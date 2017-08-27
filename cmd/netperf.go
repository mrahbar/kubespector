package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/spf13/cobra"
)

var netperfOpts = &types.NetperfOpts{}

var netperfCmd = &cobra.Command{
	Use:     "network-test",
	Aliases: []string{"net"},
	Short:   "Runs network performance tests on a cluster",
	Long:    `This is a tool for running network performance tests on a cluster. The cluster should have at least two worker nodes.`,
	PreRunE: util.CheckRequiredFlags,
	Run:     netperfRun,
}

func init() {
	PerfCmd.AddCommand(netperfCmd)
	netperfCmd.Flags().StringVarP(&netperfOpts.OutputDir, "outputDir", "o", "./netperf-results", "Full path to the directory for result files to output")
	netperfCmd.Flags().BoolVarP(&netperfOpts.Cleanup, "cleanup", "c", true, "Delete test pods when done")
}

func netperfRun(_ *cobra.Command, _ []string) {
    pkg.Netperf(createCommandContext(netperfOpts))
}
