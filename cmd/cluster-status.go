package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/spf13/cobra"
)

var clusterStatusOpts = &types.ClusterStatusOpts{}

var clusterStatusCmd = &cobra.Command{
	Use:     "cluster-status",
	Aliases: []string{"cs"},
	Short:   "Performs various checks on the cluster defined in the configuration file",
	Long:    `When called without arguments all hosts and checks in configuration will be executed.`,
	Run:     clusterStatusRun,
}

func init() {
	RootCmd.AddCommand(clusterStatusCmd)
	clusterStatusCmd.Flags().StringVarP(&clusterStatusOpts.Groups, "groups", "g", "", "Comma-separated list of group names")
	clusterStatusCmd.Flags().StringVarP(&clusterStatusOpts.Checks, "checks", "c", "", "Comma-separated list of checks. E.g. Services,Containers")
}

func clusterStatusRun(_ *cobra.Command, _ []string) {
	config := integration.UnmarshalConfig()
	clusterStatusOpts.Debug = RootOpts.Debug
	pkg.ClusterStatus(config, clusterStatusOpts)
}
