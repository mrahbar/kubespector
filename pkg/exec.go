package pkg

import (
	"fmt"
    "github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"os"
)

var execOpts *types.ExecOpts

func Exec(cmdParams *types.CommandParams) {
    initParams(cmdParams)
    execOpts = cmdParams.Opts.(*types.ExecOpts)
    runGeneric(cmdParams.Config, &execOpts.GenericOpts, initializeExec, exec)
}

func initializeExec(target string, node string, group string) {
	if group != "" {
        integration.PrintHeader(fmt.Sprintf("Executing '%v' in group [%s] ",
			target, group), '=')
	}

	if node != "" {
        integration.PrintHeader(fmt.Sprintf("Executing '%v' on node %s :\n",
			target, node), '=')
	}

	if execOpts.FileOutput != "" {
		err := util.InitializeOutputFile(execOpts.FileOutput)
		if err != nil {
            printer.PrintErr("Failed to open output file %s: %s", execOpts.FileOutput, err)
			os.Exit(1)
		} else {
            printer.PrintInfo("Result is written to file %s screen output is suppressed.", execOpts.FileOutput)
		}
	}
    integration.PrettyNewLine()
}

func exec(cmdExecutor *ssh.CommandExecutor, command string) {
	if execOpts.Sudo {
		command = fmt.Sprintf("sudo %s", command)
	} else {
		command = fmt.Sprintf("%s", command)
	}

    sshOut, err := cmdExecutor.PerformCmd(command)

    printer.Print(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))
	if err != nil {
        printer.PrintErr("Error executing command: %s", err)
	} else {
		result := ssh.CombineOutput(sshOut)
		if execOpts.FileOutput != "" {
			out := fmt.Sprintf("Result of '%s' on node %s:\n\n%s\n\n", command, util.ToNodeLabel(node), result)
			err := util.WriteOutputFile(execOpts.FileOutput, out)
			if err != nil {
                printer.PrintWarn("Failed to write to output file %s forwarding to screen: %s", execOpts.FileOutput, err)
                printer.PrintOk(result)
			} else {
                printer.PrintOk("Result written to file")
			}
		} else {
            printer.PrintOk(result)
		}
	}

    integration.PrettyNewLine()
}
