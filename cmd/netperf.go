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
	Short:   "Runs netperf tests on a cluster",
	Long:    `This is a tool for running netperf tests on a cluster. The cluster should have two worker nodes.`,
	Run:     netperfRun,
}

func init() {
	PerfCmd.AddCommand(netperfCmd)
	netperfCmd.Flags().StringVarP(&netperfOpts.Output, "output", "o", "./netperf-results.csv", "Full path to the csv file to output")
	netperfCmd.Flags().BoolVarP(&netperfOpts.Cleanup, "cleanup", "c", true, "Delete test pods when done")
}

func netperfRun(_ *cobra.Command, _ []string) {
	config := integration.UnmarshalConfig()
	netperfOpts.Debug = RootOpts.Debug
	pkg.Netperf(config, netperfOpts)
}
