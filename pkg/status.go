package pkg

import (
	"fmt"
    "github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

func Status(cmdParams *types.CommandParams) {
    initParams(cmdParams)
    opts := cmdParams.Opts.(*types.GenericOpts)
	runGeneric(config, opts, initializeStatusService, statusService)
}

func initializeStatusService(service string, node string, group string) {
	if group != "" {
        integration.PrintHeader(fmt.Sprintf("Checking status of service %v in group [%s] ",
			service, group), '=')
	}

	if node != "" {
        integration.PrintHeader(fmt.Sprintf("Checking status of service %v on node %s:",
			service, node), '=')
	}

    integration.PrettyNewLine()
}

func statusService(cmdExecutor *ssh.CommandExecutor, service string) {
    sshOut, err := cmdExecutor.PerformCmd(fmt.Sprintf("sudo systemctl status %s -l", service))

    printer.Print(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))
	if err != nil {
        printer.PrintErr("Error checking status of service %s: %s", service, err)
	} else {
        printer.PrintOk(ssh.CombineOutput(sshOut))
    }

    integration.PrettyNewLine()
}
