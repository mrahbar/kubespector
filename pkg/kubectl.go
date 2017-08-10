package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/types"
	"os"
)

func Kubectl(config types.Config, kubectlOpts *types.KubectlOpts) {
	group := integration.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		integration.PrettyPrintErr("No host configured for group [%s]", types.MASTER_GROUPNAME)
		os.Exit(1)
	}

	node := integration.GetFirstAccessibleNode(sshOpts.LocalOn, group.Nodes, kubectlOpts.Debug)

	if !integration.IsNodeAddressValid(node) {
		integration.PrettyPrintErr("No master available")
		os.Exit(1)
	}

	integration.PrettyPrint("Running kubectl command '%s' on node %s\n\n", kubectlOpts.Command, integration.ToNodeLabel(node))
	result, err := integration.PerformSSHCmd(config.Ssh, node, fmt.Sprintf("kubectl %s", kubectlOpts.Command), kubectlOpts.Debug)

	if err != nil {
		integration.PrettyPrintErr("Error performing kubectl command %s:\n\tResult: %s\tErr: %s", kubectlOpts.Command, result, err)
	} else {
		integration.PrettyPrintOk(result)
	}

	integration.PrettyPrint("\n")
}
