package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"os"
	"strings"
)

var logOpts *types.LogsOpts

func Logs(config types.Config, opts *types.LogsOpts) {
	logOpts = opts
	runGeneric(config, &opts.GenericOpts, initializeLogs, logs)
}

func initializeLogs(target string, node string, group string) {
	if group != "" {
		util.PrintHeader(fmt.Sprintf("Retrieving logs for %s %s in group [%s] ",
			logOpts.Type, target, group), '=')
	}

	if node != "" {
		util.PrintHeader(fmt.Sprintf("Retrieving logs for %s %s on node %s :\n",
			logOpts.Type, target, node), '=')
	}

	if logOpts.FileOutput != "" {
		err := util.InitializeOutputFile(logOpts.FileOutput)
		if err != nil {
			util.PrettyPrintErr("Failed to open output file %s: %s", logOpts.FileOutput, err)
			os.Exit(1)
		} else {
			util.PrettyPrintInfo("Result is written to file %s screen output is suppressed.", logOpts.FileOutput)
		}
	}
	util.PrettyNewLine()
}

func logs(sshOpts types.SSHConfig, element string, node types.Node, debug bool) {
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
	}

	cmd := fmt.Sprintf("%s", strings.Join(command, " "))

	if logOpts.Sudo {
		cmd = fmt.Sprintf("sudo %s", strings.Join(command, " "))
	}

	sshOut, err := ssh.PerformCmd(sshOpts, node, cmd, debug)

	util.PrettyPrint(fmt.Sprintf("Result on node %s:\n", util.ToNodeLabel(node)))
	if err != nil {
		util.PrettyPrintErr("Error executing command: %s", err)
	} else {
		result := ssh.CombineOutput(sshOut)
		if logOpts.FileOutput != "" {
			out := fmt.Sprintf("Result of '%s' on node %s:\n\n%s\n\n", command, util.ToNodeLabel(node), result)
			err := util.WriteOutputFile(logOpts.FileOutput, out)
			if err != nil {
				util.PrettyPrintWarn("Failed to write to output file %s forwarding to screen: %s", logOpts.FileOutput, err)
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
