package cmd

import (
	"github.com/mrahbar/kubernetes-inspector/integration"

	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/spf13/cobra"
)

var kubectlOpts = &types.KubectlOpts{}

// kubectlCmd represents the kubectl command
var kubectlCmd = &cobra.Command{
	Use:     "kubectl",
	Aliases: []string{"k"},
	Short:   "Wrapper for kubectl",
	Long:    `For a full documentation of available commands visit official website: https://kubernetes.io/docs/user-guide/kubectl-overview/`,
	PreRunE: integration.CheckRequiredFlags,
	Run:     kubectlRun,
}

func init() {
	RootCmd.AddCommand(kubectlCmd)
	kubectlCmd.Flags().StringVarP(&kubectlOpts.Command, "command", "c", "", "Command to execute")
	kubectlCmd.MarkFlagRequired("command")
}

func kubectlRun(_ *cobra.Command, _ []string) {
	config := integration.UnmarshalConfig()
	kubectlOpts.Debug = RootOpts.Debug
	pkg.Kubectl(config, kubectlOpts)
}
