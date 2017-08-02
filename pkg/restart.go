package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/types"
)

func Restart(config types.Config, opts *types.GenericOpts) {
	runGeneric(config, opts, initializeRestartService, restartService)
}

func initializeRestartService(service string, node string, group string) {
	if group != "" {
		integration.PrintHeader(fmt.Sprintf("Restarting service %v in group [%s] ",
			service, group), '=')
	}

	if node != "" {
		integration.PrintHeader(fmt.Sprintf("Restarting service %v on node %s:\n",
			service, node), '=')
	}

	integration.PrettyPrint("\n")
}

func restartService(sshOpts types.SSHConfig, service string, node types.Node, debug bool) {
	o, err := integration.PerformSSHCmd(sshOpts, node, fmt.Sprintf("sudo systemctl restart %s", service), debug)

	integration.PrettyPrint(fmt.Sprintf("Result on node %s:\n", integration.ToNodeLabel(node)))

	if err != nil {
		integration.PrettyPrintErr("Error: %v\nOut: %s", err, o)
	} else {
		integration.PrettyPrintOk("Service %s restarted. %s", service, o)
	}

	integration.PrettyPrint("\n")
}
