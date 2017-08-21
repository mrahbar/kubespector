package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"os"
)

var execOpts *types.ExecOpts

func Exec(config types.Config, opts *types.ExecOpts) {
	execOpts = opts
	runGeneric(config, &opts.GenericOpts, initializeExec, exec)
}

func initializeExec(target string, node string, group string) {
	if group != "" {
		util.PrintHeader(fmt.Sprintf("Executing '%v' in group [%s] ",
			target, group), '=')
	}

	if node != "" {
		util.PrintHeader(fmt.Sprintf("Executing '%v' on node %s :\n",
			target, node), '=')
	}

	if execOpts.FileOutput != "" {
		err := util.InitializeOutputFile(execOpts.FileOutput)
		if err != nil {
			util.PrettyPrintErr("Failed to open output file %s: %s", execOpts.FileOutput, err)
			os.Exit(1)
		} else {
			util.PrettyPrintInfo("Result is written to file %s screen output is suppressed.", execOpts.FileOutput)
		}
	}
	util.PrettyNewLine()
}

func exec(sshOpts types.SSHConfig, command string, node types.Node, debug bool) {
	if execOpts.Sudo {
		command = fmt.Sprintf("sudo %s", command)
	} else {
		command = fmt.Sprintf("%s", command)
	}

	sshOut, err := ssh.PerformCmd(sshOpts, node, command, debug)

	util.PrettyPrint(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))
	if err != nil {
		util.PrettyPrintErr("Error executing command: %s", err)
	} else {
		result := ssh.CombineOutput(sshOut)
		if execOpts.FileOutput != "" {
			out := fmt.Sprintf("Result of '%s' on node %s:\n\n%s\n\n", command, util.ToNodeLabel(node), result)
			err := util.WriteOutputFile(execOpts.FileOutput, out)
			if err != nil {
				util.PrettyPrintWarn("Failed to write to output file %s forwarding to screen: %s", execOpts.FileOutput, err)
				util.PrettyPrintOk(result)
			} else {
				util.PrettyPrintOk("Result written to file")
			}
		} else {
			util.PrettyPrintOk(result)
		}
	}

	util.PrettyNewLine()
}
