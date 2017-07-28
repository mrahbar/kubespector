package cmd

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/integration"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

type kubectlCliOpts struct {
	command string
}

var kubectlOpts = &kubectlCliOpts{}

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
	kubectlCmd.Flags().StringVarP(&kubectlOpts.command, "command", "c", "", "Command to execute")
	kubectlCmd.MarkFlagRequired("command")
}

func kubectlRun(_ *cobra.Command, _ []string) {
	var config integration.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		integration.PrettyPrintErr(out, "Unable to decode config: %v", err)
		os.Exit(1)
	}

	group := integration.FindGroupByName(config.ClusterGroups, integration.MASTER_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		integration.PrettyPrintErr(out, "No host configured for group [%s]", integration.MASTER_GROUPNAME)
		os.Exit(1)
	}

	node := integration.GetFirstAccessibleNode(group.Nodes, RootOpts.Debug)

	if !integration.IsNodeAddressValid(node) {
		integration.PrettyPrintErr(out, "No master available")
		os.Exit(1)
	}

	integration.PrettyPrint(out, "Running kubectl command '%s' on node %s\n\n", kubectlOpts.command, integration.ToNodeLabel(node))
	o, err := integration.PerformSSHCmd(out, config.Ssh, node, fmt.Sprintf("kubectl %s", kubectlOpts.command), RootOpts.Debug)
	result := strings.TrimSpace(o)

	if err != nil {
		integration.PrettyPrintErr(out, "Error performing kubectl command %s:\n\tResult: %s\tErr: %s", kubectlOpts.command, result, err)
	} else {
		integration.PrettyPrintOk(out, result)
	}

	integration.PrettyPrint(out, "\n")
}
