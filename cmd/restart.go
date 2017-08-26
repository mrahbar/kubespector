package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/util"

	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/spf13/cobra"
)

var restartOpts = &types.GenericOpts{}

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restarts a Kubernetes service on a target group or node",
	Long: `Service name is mandatory. Either specify node or group in which the service should be restarted.
	When a target group is specified all nodes inside that group will be targeted for service restart.`,
	PreRunE: util.CheckRequiredFlags,
	Run:     restartRun,
}

func init() {
	ServiceCmd.AddCommand(restartCmd)
	restartCmd.Flags().StringVarP(&restartOpts.GroupArg, "group", "g", "", "Comma-separated list of group names")
	restartCmd.Flags().StringVarP(&restartOpts.NodeArg, "node", "n", "", "Name of target node")
	restartCmd.Flags().StringVarP(&restartOpts.TargetArg, "service", "s", "", "Name of target service")
	restartCmd.MarkFlagRequired("service")
}

func restartRun(_ *cobra.Command, _ []string) {
	config := util.UnmarshalConfig()
    params := &types.CommandParams{
        Printer: printer,
        Config:  config,
        Opts:    restartOpts,
    }
    pkg.Restart(params)
}
