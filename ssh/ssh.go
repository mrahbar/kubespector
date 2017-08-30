package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"bytes"
    "github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/ssh/communicator"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"golang.org/x/crypto/ssh"
)

type Executor struct {
    SshOpts types.SSHConfig
    Node    types.Node
    Printer integration.LogWriter
}

func (c *Executor) SetNode(node types.Node) {
    c.Node = node
}

func (c *Executor) PerformCmd(cmd string) (*types.SSHOutput, error) {
    if util.NodeEquals(c.SshOpts.LocalOn, c.Node) {
        return shell(cmd, c.Printer)
    }

    comm, err := establishSSHCommunication(c.SshOpts, util.GetNodeAddress(c.Node), c.Printer)
	if err != nil {
        c.Printer.PrintDebug("Creating communicator failed: %s", err)
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
        c.Printer.PrintDebug("Starting remote command failed: %s", err)
		return &types.SSHOutput{}, err
	}
	remoteCmd.Wait()
	output := strings.TrimSpace(stdout.String())
	outErr := strings.TrimSpace(stderr.String())
	o := &types.SSHOutput{Stdout: output, Stderr: outErr}

    if remoteCmd.ExitStatus != 0 {
        err = fmt.Errorf("%s", outErr)
    }

    errFormatted := ""
    if err != nil {
        errFormatted = fmt.Sprintf("%s", err)
    }
    c.Printer.PrintDebug("Result of command '%s':\nStdout: %s\nStderr: %s\nExitStatus: %d\nErr: %s\n",
        cmd, output, outErr, remoteCmd.ExitStatus, errFormatted)

	return o, err
}

func shell(cmd string, printer integration.LogWriter) (*types.SSHOutput, error) {
	shell := "/bin/bash"
    err := findExecutable(shell)
	if err != nil {
		shell = "/bin/sh"
        err := findExecutable(shell)
		if err != nil {
			return &types.SSHOutput{}, err
		}
	}

	execCmd := exec.Command(shell, "-c", cmd)

    printer.PrintDebug("Executing command: %s %s\n", execCmd.Path, execCmd.Args)

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

    errFormatted := ""
    if err != nil {
        errFormatted = fmt.Sprintf("%s", err)
    }
    printer.PrintDebug("Result of command\n- Stdout: %s\n- Stderr: %s\n- ExitStatus: %d\n- Err: %s\n",
        output, outErr, exitStatus, errFormatted)

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
