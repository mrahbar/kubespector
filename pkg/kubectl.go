package pkg

import (
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"strings"
)

func Kubectl(cmdParams *types.CommandContext) {
	initParams(cmdParams)
	kubectlOpts := cmdParams.Opts.(*types.KubectlOpts)
	group := util.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		printer.PrintCritical("No host configured for group [%s]", types.MASTER_GROUPNAME)
	}

	node := ssh.GetFirstAccessibleNode(config.Ssh.LocalOn, cmdExecutor, group.Nodes)

	if !util.IsNodeAddressValid(node) {
		printer.PrintCritical("No master available")
	}

	cmdExecutor.SetNode(node)
	printer.Print("Running kubectl command '%s' on node %s\n", kubectlOpts.Command, util.ToNodeLabel(node))
	sshOut, err := cmdExecutor.RunKubectlCommand(strings.Split(kubectlOpts.Command, " "))

	if err != nil {
		printer.PrintErr("Error performing kubectl command '%s': %s", kubectlOpts.Command, err)
	} else {
		printer.PrintOk(sshOut.Stdout)
	}
}
