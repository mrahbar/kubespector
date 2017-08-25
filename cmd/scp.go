package cmd

import (
	"github.com/spf13/cobra"
	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
)

var scpOpts = &types.ScpOpts{}

var scpCmd = &cobra.Command{
	Use:     "scp",
	Short:   "Secure bidirectional file copy",
	Long:    `Copies files or directories to or from the cluster`,
	PreRunE: util.CheckRequiredFlags,
	Run:     scpRun,
}

func init() {
	RootCmd.AddCommand(scpCmd)
	scpCmd.Flags().StringVarP(&scpOpts.GroupArg, "group", "g", "", "Comma-separated list of group names")
	scpCmd.Flags().StringVarP(&scpOpts.NodeArg, "node", "n", "", "Name of target node")
	scpCmd.Flags().StringVarP(&scpOpts.TargetArg, "direction", "t", "", "Must either be 'up' or 'down' resp. first letter.")
	scpCmd.Flags().StringVarP(&scpOpts.LocalPath, "localPath", "l", "", "This is the source when direction is 'up' or the target when direction is 'down' ")
	scpCmd.Flags().StringVarP(&scpOpts.RemotePath, "remotePath", "r", "", "This is the target when direction is 'up' or the source when direction is 'down' ")

	scpCmd.MarkFlagRequired("direction")
	scpCmd.MarkFlagRequired("localPath")
	scpCmd.MarkFlagRequired("remotePath")
}

func scpRun(_ *cobra.Command, _ []string) {
	config := util.UnmarshalConfig()
	scpOpts.Debug = RootOpts.Debug
	pkg.Scp(config, scpOpts)
}