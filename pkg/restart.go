package pkg

import (
	"fmt"
    "github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

func Restart(cmdParams *types.CommandParams) {
    initParams(cmdParams)
    opts := cmdParams.Opts.(*types.GenericOpts)
	runGeneric(config, opts, initializeRestartService, restartService)
}

func initializeRestartService(service string, node string, group string) {
	if group != "" {
        integration.PrintHeader(fmt.Sprintf("Restarting service %v in group [%s] ",
			service, group), '=')
	}

	if node != "" {
        integration.PrintHeader(fmt.Sprintf("Restarting service %v on node %s:",
			service, node), '=')
	}

    integration.PrettyNewLine()
}

func restartService(cmdExecutor *ssh.CommandExecutor, service string) {
    _, err := cmdExecutor.PerformCmd(fmt.Sprintf("sudo systemctl restart %s", service))

    printer.Print(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))

	if err != nil {
        printer.PrintErr("Error restarting service %s: %s", service, err)
	} else {
        printer.PrintOk("Service %s restarted.", service)
    }

    integration.PrettyNewLine()
}
