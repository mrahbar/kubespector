package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/spf13/cobra"
)

var scaleTestOpts = &types.ScaleTestOpts{}

// scaleCmd represents the scale command
var scaleCmd = &cobra.Command{
	Use:     "scale-test",
	Aliases: []string{"scale"},
	Short:   "Runs a load tests on a cluster",
	Long:    `This is a tool for running a scale test on a cluster by performing massive load on network and on pods.`,
	PreRunE: util.CheckRequiredFlags,
	Run:     scaleRun,
}

func init() {
	PerfCmd.AddCommand(scaleCmd)
    scaleCmd.Flags().StringVarP(&scaleTestOpts.OutputDir, "output", "o", "scaletest-results", "Full path to directory for result files")
	scaleCmd.Flags().BoolVarP(&scaleTestOpts.Cleanup, "cleanup", "c", true, "Delete test pods when done")
}

func scaleRun(_ *cobra.Command, _ []string) {
	config := util.UnmarshalConfig()
    params := &types.CommandParams{
        Printer: printer,
        Config:  config,
        Opts:    scaleTestOpts,
    }
    pkg.ScaleTest(params)
}
