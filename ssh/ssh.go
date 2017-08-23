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
	"golang.org/x/crypto/ssh"
)

func PerformCmd(sshOpts types.SSHConfig, node types.Node, cmd string, debug bool) (*types.SSHOutput, error) {
	if util.NodeEquals(sshOpts.LocalOn, node) {
		splits := strings.SplitN(cmd, " ", 1)
		var args []string
		if len(splits) > 1 {
			args = strings.Split(splits[1], " ")
		} else {
			args = []string{}
		}
		out, err := shell(splits[0], debug, args...)
		return out, err
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
func shell(binaryPath string, debug bool, args ...string) (*types.SSHOutput, error) {
	cmd := exec.Command(binaryPath, args...)
	if debug {
		cmdDebug := append([]string{}, cmd.Args...)
		fmt.Printf("Executing command: %s\n", cmdDebug)
	}

	var stderr bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	output := strings.TrimSpace(string(out))
	outErr := strings.TrimSpace(stderr.String())
	o := &types.SSHOutput{Stdout: output, Stderr: outErr}

	exitStatus := 0
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			exitStatus = err.(*ssh.ExitError).ExitStatus()
		}
	}

	if debug {
		errFormatted := ""
		if err != nil {
			errFormatted = fmt.Sprintf("%s", err)
		}
		util.PrettyPrintDebug("Result of command:\nStdout: %s\nStderr: %s\nExitStatus: %d\nErr: %s\n",
			output, outErr, exitStatus, errFormatted)
	}

	return o, err
}
