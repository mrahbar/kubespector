package pkg

import (
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"os"
	"strings"
)

func Kubectl(config types.Config, kubectlOpts *types.KubectlOpts) {
	group := util.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		util.PrettyPrintErr("No host configured for group [%s]", types.MASTER_GROUPNAME)
		os.Exit(1)
	}

	node := ssh.GetFirstAccessibleNode(config.Ssh, group.Nodes, kubectlOpts.Debug)

	if !util.IsNodeAddressValid(node) {
		util.PrettyPrintErr("No master available")
		os.Exit(1)
	}

	util.PrettyPrint("Running kubectl command '%s' on node %s\n", kubectlOpts.Command, util.ToNodeLabel(node))
	sshOut, err := ssh.RunKubectlCommand(config.Ssh, node, strings.Split(kubectlOpts.Command, " "), kubectlOpts.Debug)

	if err != nil {
		util.PrettyPrintErr("Error performing kubectl command %s: %s", kubectlOpts.Command, err)
	} else {
		util.PrettyPrintOk(sshOut.Stdout)
	}
}
