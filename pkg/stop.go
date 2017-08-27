package pkg

import (
	"fmt"
    "github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

func Stop(cmdParams *types.CommandContext) {
    initParams(cmdParams)
    opts := cmdParams.Opts.(*types.GenericOpts)
	runGeneric(config, opts, initializeStopService, stopService)
}

func initializeStopService(service string, node string, group string) {
	if group != "" {
        integration.PrintHeader(fmt.Sprintf("Stopping service %v in group [%s] ",
			service, group), '=')
	}

	if node != "" {
        integration.PrintHeader(fmt.Sprintf("Stopping service %v on node %s:",
			service, node), '=')
	}

    integration.PrettyNewLine()
}

func stopService(service string) {
    _, err := cmdExecutor.PerformCmd(fmt.Sprintf("sudo systemctl stop %s", service))

    printer.Print(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))
	if err != nil {
        printer.PrintErr("Error stopping service %s: %s", service, err)
	} else {
        printer.PrintOk("Service %s stopped.", service)
    }

    integration.PrettyNewLine()
}
