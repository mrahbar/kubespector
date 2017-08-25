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
		return shell(cmd, debug)
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

func shell(cmd string, debug bool) (*types.SSHOutput, error) {
	shell := "/bin/bash"
	err := findExecutable(shell);
	if err != nil {
		shell = "/bin/sh"
		err := findExecutable(shell);
		if err != nil {
			return &types.SSHOutput{}, err
		}
	}

	execCmd := exec.Command(shell, "-c", cmd)

	if debug {
		fmt.Printf("Executing command: %s %s\n", execCmd.Path, execCmd.Args)
	}

	var stderr bytes.Buffer
	execCmd.Stdin = os.Stdin
	execCmd.Stderr = &stderr
	out, err := execCmd.Output()
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
		util.PrettyPrintDebug("Result of command\n- Stdout: %s\n- Stderr: %s\n- ExitStatus: %d\n- Err: %s\n",
			output, outErr, exitStatus, errFormatted)
	}

	return o, err
}

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}