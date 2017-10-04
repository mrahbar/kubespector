package pkg

import (
	"fmt"

	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

var restartOpts *types.GenericOpts

func Restart(cmdParams *types.CommandContext) {
    initParams(cmdParams)
    restartOpts = cmdParams.Opts.(*types.GenericOpts)
	runGeneric(config, restartOpts, initializeRestartService, restartService)
}

func initializeRestartService(service string, node string) {
    printer.PrintHeader(fmt.Sprintf("Restarting service %v on node %s",
        service, node), '=')
    printer.PrintNewLine()
}

func restartService(service string) {
    _, err := cmdExecutor.PerformCmd(fmt.Sprintf("systemctl restart %s", service), restartOpts.Sudo)

    printer.Print(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(cmdExecutor.GetNode())))

	if err != nil {
        printer.PrintErr("Error restarting service %s: %s", service, err)
	} else {
        printer.PrintOk("Service %s restarted.", service)
    }

    printer.PrintNewLine()
}
