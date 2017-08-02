package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/types"
)

func Exec(config types.Config, opts *types.ExecOpts) {
	runGeneric(config, &opts.GenericOpts, initializeExec, exec)
}

func initializeExec(target string, node string, group string) {
	if group != "" {
		integration.PrintHeader(fmt.Sprintf("Executing '%v' in group [%s] ",
			target, group), '=')
	}

	if node != "" {
		integration.PrintHeader(fmt.Sprintf("Executing '%v' on node %s :\n",
			target, node), '=')
	}

	integration.PrettyPrint("\n")
}

func exec(sshOpts types.SSHConfig, command string, node types.Node, debug bool) {
	command = fmt.Sprintf("bash -c '%s'", command)

	o, err := integration.PerformSSHCmd(sshOpts, node, command, debug)

	integration.PrettyPrint(fmt.Sprintf("Result on node %s:\n", integration.ToNodeLabel(node)))
	if err != nil {
		integration.PrettyPrintErr("Error: %v\nOut: %s", err, o)
	} else {
		integration.PrettyPrintOk("%s", o)
	}

	integration.PrettyPrint("\n")
}
