package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/types"
)

func Stop(config types.Config, opts *types.GenericOpts) {
	runGeneric(config, opts, initializeStopService, stopService)
}

func initializeStopService(service string, node string, group string) {
	if group != "" {
		integration.PrintHeader(fmt.Sprintf("Stopping service %v in group [%s] ",
			service, group), '=')
	}

	if node != "" {
		integration.PrintHeader(fmt.Sprintf("Stopping service %v on node %s:\n",
			service, node), '=')
	}

	integration.PrettyPrint("\n")
}

func stopService(sshOpts types.SSHConfig, service string, node types.Node, debug bool) {
	o, err := integration.PerformSSHCmd(sshOpts, node, fmt.Sprintf("sudo systemctl stop %s", service), debug)

	integration.PrettyPrint(fmt.Sprintf("Result on node %s:\n", integration.ToNodeLabel(node)))
	if err != nil {
		integration.PrettyPrintErr("Error: %v\nOut: %s", err, o)
	} else {
		integration.PrettyPrintOk("Service %s stoped. %s", service, o)
	}

	integration.PrettyPrint("\n")
}
