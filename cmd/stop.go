package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/util"

	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/spf13/cobra"
)

var stopOpts = &types.GenericOpts{}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
    Short: "Stop a system service on a target group or node",
	Long: `Service name is mandatory. Either specify node or group in which the service should be stoped.
	When a target group is specified all nodes inside that group will be targeted for service stop.`,
	PreRunE: util.CheckRequiredFlags,
	Run:     stopRun,
}

func init() {
	ServiceCmd.AddCommand(stopCmd)
	stopCmd.Flags().StringVarP(&stopOpts.GroupArg, "group", "g", "", "Comma-separated list of group names")
	stopCmd.Flags().StringVarP(&stopOpts.NodeArg, "node", "n", "", "Name of target node")
	stopCmd.Flags().StringVarP(&stopOpts.TargetArg, "service", "s", "", "Name of target service")
	stopCmd.MarkFlagRequired("service")
}

func stopRun(_ *cobra.Command, _ []string) {
    pkg.Stop(createCommandContext(stopOpts))
}
