package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"bytes"
	"github.com/mrahbar/kubernetes-inspector/ssh/communicator"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

func PerformCmd(sshOpts types.SSHConfig, node types.Node, cmd string, debug bool) (*types.SSHOutput, error) {
	if util.NodeEquals(sshOpts.LocalOn, node) {
		splits := strings.SplitN(cmd, " ", 1)
		args := ""
		if len(splits) > 1 {
			args = splits[1]
		}
		out, err := shell(splits[0], debug, args)
		return &types.SSHOutput{Stdout: out}, err
	}

	comm, err := establishSSHCommunication(sshOpts, util.GetNodeAddress(node), debug)
	if err != nil {
		if debug {
			util.PrettyPrintDebug("Creating communicator failed: %s", err)
		}
		return &types.SSHOutput{}, err
	}

	var stdout, stderr bytes.Buffer
	remoteCmd := &communicator.RemoteCmd{
		Command: cmd,
		Stdin:   os.Stdin,
		Stdout:  &stdout,
		Stderr:  &stderr,
	}

	err = comm.Start(remoteCmd)
	if err != nil {
		if debug {
			util.PrettyPrintDebug("Starting remote command failed: %s", err)
		}
		return &types.SSHOutput{}, err
	}
	remoteCmd.Wait()
	output := strings.TrimSpace(stdout.String())
	outErr := strings.TrimSpace(stderr.String())
	o := &types.SSHOutput{Stdout: output, Stderr: outErr}

	if debug {
		errFormatted := ""
		if err != nil {
			errFormatted = fmt.Sprintf("%s", err)
		}
		util.PrettyPrintDebug("Result of command:\nStdout: %s\nStderr: %s\nExitStatus: %d\nErr: %s\n",
			output, outErr, remoteCmd.ExitStatus, errFormatted)
	}

	return o, err
}

// Shell runs the command, binding Stdin, Stdout and Stderr
func shell(binaryPath string, debug bool, args ...string) (string, error) {
	cmd := exec.Command(binaryPath, args...)
	if debug {
		cmdDebug := append([]string{}, cmd.Args...)
		fmt.Printf("Executing command: %s\n", cmdDebug)
	}

	cmd.Stdin = os.Stdin
	o, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(o))
	if debug {
		fmt.Printf("Result of command:\nResult: %sErr: %s\n", output, err)
	}

	return output, err
}
