package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/spf13/cobra"
)

var clusterStatusOpts = &types.ClusterStatusOpts{}

var clusterStatusCmd = &cobra.Command{
	Use:     "cluster-status",
	Aliases: []string{"cs"},
	Short:   "Performs various checks on the cluster defined in the configuration file",
	Long:    `When called without arguments all hosts and checks in configuration will be executed.`,
	PreRunE: util.CheckRequiredFlags,
	Run:     clusterStatusRun,
}

func init() {
	RootCmd.AddCommand(clusterStatusCmd)
	clusterStatusCmd.Flags().StringVarP(&clusterStatusOpts.Groups, "groups", "g", "", "Comma-separated list of group names")
	clusterStatusCmd.Flags().StringVarP(&clusterStatusOpts.Checks, "checks", "c", "", "Comma-separated list of checks. E.g. Services,Containers")
}

func clusterStatusRun(_ *cobra.Command, _ []string) {
	config := util.UnmarshalConfig()
    params := &types.CommandParams{
        Printer: printer,
        Config:  config,
        Opts:    clusterStatusOpts,
    }
    pkg.ClusterStatus(params)
}
