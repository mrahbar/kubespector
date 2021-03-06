package pkg

import (
	"fmt"

	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

var stopOpts *types.GenericOpts

func Stop(cmdParams *types.CommandContext) {
    initParams(cmdParams)
    stopOpts = cmdParams.Opts.(*types.GenericOpts)
	runGeneric(config, stopOpts, initializeStopService, stopService)
}

func initializeStopService(service string, node string) {
    printer.PrintHeader(fmt.Sprintf("Stopping service %v on node %s", service, node), '=')
    printer.PrintNewLine()
}

func stopService(service string) {
    _, err := cmdExecutor.PerformCmd(fmt.Sprintf("systemctl stop %s", service), stopOpts.Sudo)

    printer.Print(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(cmdExecutor.GetNode())))
	if err != nil {
        printer.PrintErr("Error stopping service %s: %s", service, err)
	} else {
        printer.PrintOk("Service %s stopped.", service)
    }

    printer.PrintNewLine()
}
