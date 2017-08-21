package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

func Status(config types.Config, opts *types.GenericOpts) {
	runGeneric(config, opts, initializeStatusService, statusService)
}

func initializeStatusService(service string, node string, group string) {
	if group != "" {
		util.PrintHeader(fmt.Sprintf("Checking status of service %v in group [%s] ",
			service, group), '=')
	}

	if node != "" {
		util.PrintHeader(fmt.Sprintf("Checking status of service %v on node %s:",
			service, node), '=')
	}

	util.PrettyNewLine()
}

func statusService(sshOpts types.SSHConfig, service string, node types.Node, debug bool) {
	sshOut, err := ssh.PerformCmd(sshOpts, node, fmt.Sprintf("sudo systemctl status %s -l", service), debug)

	util.PrettyPrint(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))
	if err != nil {
		util.PrettyPrintErr("Error checking status of service %s: %s", service, err)
	} else {
		util.PrettyPrintOk(ssh.CombineOutput(sshOut))
	}

	util.PrettyNewLine()
}
