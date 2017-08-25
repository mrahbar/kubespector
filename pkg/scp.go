package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

var scpOpts *types.ScpOpts

func Scp(config types.Config, opts *types.ScpOpts) {
	scpOpts = opts
	runGeneric(config, &opts.GenericOpts, initializeScp, scp)
}

func initializeScp(target string, node string, group string) {
	if group != "" {
		util.PrintHeader(fmt.Sprintf("Executing scp in group [%s] ", group), '=')
	}

	if node != "" {
		util.PrintHeader(fmt.Sprintf("Executing scp on node %s :\n", node), '=')
	}

	util.PrettyNewLine()
}

func scp(sshOpts types.SSHConfig, command string, node types.Node, debug bool) {
	command = fmt.Sprintf("%s", command)
	sshOut, err := ssh.PerformCmd(sshOpts, node, command, debug)

	util.PrettyPrint(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))
	if err != nil {
		util.PrettyPrintErr("Error executing command: %s", err)
	} else {
		result := ssh.CombineOutput(sshOut)
		if scpOpts.LocalPath != "" {
			out := fmt.Sprintf("Result of '%s' on node %s:\n\n%s\n\n", command, util.ToNodeLabel(node), result)
			err := util.WriteOutputFile(scpOpts.LocalPath, out)
			if err != nil {
				util.PrettyPrintWarn("Failed to write to output file %s forwarding to screen: %s", scpOpts.LocalPath, err)
				util.PrettyPrintOk(result)
			} else {
				util.PrettyPrintOk("Result written to file")
			}
		} else {
			util.PrettyPrintOk(result)
		}
	}

	util.PrettyNewLine()
}
