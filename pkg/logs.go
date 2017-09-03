package pkg

import (
	"fmt"

	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"strings"
)

var logOpts *types.LogsOpts

func Logs(cmdParams *types.CommandContext) {
	initParams(cmdParams)
	logOpts = cmdParams.Opts.(*types.LogsOpts)
	runGeneric(cmdParams.Config, &logOpts.GenericOpts, initializeLogs, logs)
}

func initializeLogs(target string, node string, group string) {
	if group != "" {
		printer.PrintHeader(fmt.Sprintf("Retrieving logs for %s %s in group [%s] ",
			logOpts.Type, target, group), '=')
	}

	if node != "" {
		printer.PrintHeader(fmt.Sprintf("Retrieving logs for %s %s on node %s :\n",
			logOpts.Type, target, node), '=')
	}

	if logOpts.FileOutput != "" {
		err := util.InitializeOutputFile(logOpts.FileOutput)
		if err != nil {
			printer.PrintCritical("Failed to open output file %s: %s", logOpts.FileOutput, err)
		} else {
			printer.PrintInfo("Result is written to file %s screen output is suppressed.", logOpts.FileOutput)
		}
	}
	printer.PrintNewLine()
}

func logs(element string) {
	command := []string{}
	switch logOpts.Type {
	case "service":
		command = append(command, "journalctl")
		if logOpts.Tail > 0 {
			command = append(command, fmt.Sprintf("--lines=%d", logOpts.Tail))
		}
		if logOpts.Since != "" {
			command = append(command, fmt.Sprintf("--since=%s", logOpts.Since))
		}
		if len(logOpts.ExtraArgs) > 0 {
			for _, arg := range logOpts.ExtraArgs {
				command = append(command, arg)
			}
		}
		command = append(command, fmt.Sprintf("--unit=%s", element))
	case "container":
		command = append(command, "docker logs")
		if logOpts.Tail > 0 {
			command = append(command, fmt.Sprintf("--tail %d", logOpts.Tail))
		}
		if logOpts.Since != "" {
			command = append(command, fmt.Sprintf("--since %s", logOpts.Since))
		}
		if len(logOpts.ExtraArgs) > 0 {
			for _, arg := range logOpts.ExtraArgs {
				command = append(command, arg)
			}
		}
		command = append(command, element)
	case "pod":
		command = append(command, "kubectl logs")
		if logOpts.Tail > 0 {
			command = append(command, fmt.Sprintf("--tail=%d", logOpts.Tail))
		}
		if logOpts.Since != "" {
			command = append(command, fmt.Sprintf("--since=%s", logOpts.Since))
		}
		if len(logOpts.ExtraArgs) > 0 {
			for _, arg := range logOpts.ExtraArgs {
				command = append(command, arg)
			}
		}
		command = append(command, element)
	default:
		printer.PrintCritical("Unknown type %s", logOpts.Type)
	}

	cmd := fmt.Sprintf("%s", strings.Join(command, " "))

	if logOpts.Sudo {
		cmd = fmt.Sprintf("sudo %s", strings.Join(command, " "))
	}

	sshOut, err := cmdExecutor.PerformCmd(cmd)

	printer.Print(fmt.Sprintf("Result on node %s:\n", util.ToNodeLabel(cmdExecutor.GetNode())))
	if err != nil {
		printer.PrintErr("Error executing command: %s", err)
	} else {
		result := ssh.CombineOutput(sshOut)
		if logOpts.FileOutput != "" {
			out := fmt.Sprintf("Result of '%s' on node %s:\n\n%s\n\n", strings.Join(command, " "), util.ToNodeLabel(cmdExecutor.GetNode()), result)
			err := util.WriteOutputFile(logOpts.FileOutput, out)
			if err != nil {
				printer.PrintWarn("Failed to write to output file %s forwarding to screen: %s", logOpts.FileOutput, err)
				printer.PrintOk(result)
			} else {
				printer.PrintOk("Result written to file")
			}
		} else {
			printer.PrintOk(result)
		}
	}

	printer.PrintNewLine()
}
