package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

func Stop(config types.Config, opts *types.GenericOpts) {
	runGeneric(config, opts, initializeStopService, stopService)
}

func initializeStopService(service string, node string, group string) {
	if group != "" {
		util.PrintHeader(fmt.Sprintf("Stopping service %v in group [%s] ",
			service, group), '=')
	}

	if node != "" {
		util.PrintHeader(fmt.Sprintf("Stopping service %v on node %s:",
			service, node), '=')
	}

	util.PrettyNewLine()
}

func stopService(sshOpts types.SSHConfig, service string, node types.Node, debug bool) {
	_, err := ssh.PerformCmd(sshOpts, node, fmt.Sprintf("sudo systemctl stop %s", service), debug)

	util.PrettyPrint(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))
	if err != nil {
		util.PrettyPrintErr("Error stopping service %s: %s", service, err)
	} else {
		util.PrettyPrintOk("Service %s stopped.", service)
	}

	util.PrettyNewLine()
}
