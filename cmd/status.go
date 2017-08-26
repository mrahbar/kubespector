package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/util"

	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/spf13/cobra"
)

var statusOpts = &types.GenericOpts{}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of a service on a target group or node",
	Long: `Service name is mandatory. Either specify node or group in which the service status should be checked.
	When a target group is specified all nodes inside that group will be targeted.`,
	PreRunE: util.CheckRequiredFlags,
	Run:     statusRun,
}

func init() {
	ServiceCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVarP(&statusOpts.GroupArg, "group", "g", "", "Comma-separated list of group names")
	statusCmd.Flags().StringVarP(&statusOpts.NodeArg, "node", "n", "", "Name of target node")
	statusCmd.Flags().StringVarP(&statusOpts.TargetArg, "service", "s", "", "Name of target service")
	statusCmd.MarkFlagRequired("service")
}

func statusRun(_ *cobra.Command, _ []string) {
	config := util.UnmarshalConfig()
	params := &types.CommandParams{
		Printer: printer,
		Config:  config,
		Opts:    statusOpts,
	}
	pkg.Status(params)
}
