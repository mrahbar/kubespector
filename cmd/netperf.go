package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/spf13/cobra"
)

var netperfOpts = &types.NetperfOpts{}

var netperfCmd = &cobra.Command{
	Use:     "network",
	Aliases: []string{"net"},
	Short:   "Runs network performance tests on a cluster",
	Long:    `This is a tool for running network performance tests on a cluster. The cluster should have at least two worker nodes.`,
	Run:     netperfRun,
}

func init() {
	PerfCmd.AddCommand(netperfCmd)
	netperfCmd.Flags().StringVarP(&netperfOpts.OutputDir, "outputDir", "o", "./netperf-results", "Full path to the directory for result files to output")
	netperfCmd.Flags().BoolVarP(&netperfOpts.Cleanup, "cleanup", "c", true, "Delete test pods when done")
}

func netperfRun(_ *cobra.Command, _ []string) {
	config := integration.UnmarshalConfig()
	netperfOpts.Debug = RootOpts.Debug
	pkg.Netperf(config, netperfOpts)
}
