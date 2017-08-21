package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

func Restart(config types.Config, opts *types.GenericOpts) {
	runGeneric(config, opts, initializeRestartService, restartService)
}

func initializeRestartService(service string, node string, group string) {
	if group != "" {
		util.PrintHeader(fmt.Sprintf("Restarting service %v in group [%s] ",
			service, group), '=')
	}

	if node != "" {
		util.PrintHeader(fmt.Sprintf("Restarting service %v on node %s:",
			service, node), '=')
	}

	util.PrettyNewLine()
}

func restartService(sshOpts types.SSHConfig, service string, node types.Node, debug bool) {
	_, err := ssh.PerformCmd(sshOpts, node, fmt.Sprintf("sudo systemctl restart %s", service), debug)

	util.PrettyPrint(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))

	if err != nil {
		util.PrettyPrintErr("Error restarting service %s: %s", service, err)
	} else {
		util.PrettyPrintOk("Service %s restarted.", service)
	}

	util.PrettyNewLine()
}
