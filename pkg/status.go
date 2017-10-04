package pkg

import (
	"fmt"

	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

var statusOpts *types.GenericOpts

func Status(cmdParams *types.CommandContext) {
    initParams(cmdParams)
    statusOpts = cmdParams.Opts.(*types.GenericOpts)
	runGeneric(config, statusOpts, initializeStatusService, statusService)
}

func initializeStatusService(service string, node string) {
    printer.PrintHeader(fmt.Sprintf("Checking status of service %v on node %s",
        service, node), '=')
    printer.PrintNewLine()
}

func statusService(service string) {
    sshOut, err := cmdExecutor.PerformCmd(fmt.Sprintf("systemctl status %s -l", service), statusOpts.Sudo)

    printer.Print(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(cmdExecutor.GetNode())))
	if err != nil {
        printer.PrintErr("Error checking status of service %s: %s", service, err)
	} else {
        printer.PrintOk(ssh.CombineOutput(sshOut))
    }

    printer.PrintNewLine()
}
