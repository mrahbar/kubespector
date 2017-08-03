package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/spf13/cobra"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/pkg"
)

var netperfOpts = &types.NetperfOpts{}

var netperfCmd = &cobra.Command{
	Use:   "net",
	Short: "Runs netperf tests on a cluster",
	Long:  `This is a tool for running netperf tests on a cluster. The cluster should have two worker nodes.`,
	Run:   netperfRun,
}

func init() {
	PerfCmd.AddCommand(netperfCmd)
	netperfCmd.Flags().StringVarP(&netperfOpts.Output, "output", "o", "./netperf.out", "Full path to the csv file to output")
	netperfCmd.Flags().BoolVarP(&netperfOpts.Cleanup, "cleanup", "c", true, "Delete test pods when done")
}

func netperfRun(_ *cobra.Command, _ []string) {
	config := integration.UnmarshalConfig()
	netperfOpts.Debug = RootOpts.Debug
	pkg.Netperf(config, netperfOpts)
}