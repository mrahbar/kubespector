package integration

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/appleboy/easyssh-proxy"
	"time"
)

const connectionTimeout = 10 * time.Second
const commandTimeout = 15

func PerformSSHCmd(sshOpts types.SSHConfig, node types.Node, cmd string, debug bool) (string, error) {
	if NodeEquals(sshOpts.LocalOn, node) {
		splits := strings.SplitN(cmd, " ", 1)
		return shell(splits[0], debug, splits[1])
	}

	nodeAddress := GetNodeAddress(node)
	key, err := validUnencryptedPrivateKey(sshOpts.Connection.Key, debug)
	if err != nil {
		return "", err
	}

	sshConf := &easyssh.MakeConfig{
		Port:    fmt.Sprintf("%d", sshOpts.Connection.Port),
		User:    sshOpts.Connection.User,
		KeyPath: key,
		Server:  nodeAddress,
		Timeout: connectionTimeout,
	}

	if IsNodeAddressValid(sshOpts.Bastion.Node) {
		sshConf.Proxy = easyssh.DefaultConfig{
			Server:  GetNodeAddress(sshOpts.Bastion.Node),
			KeyPath: sshOpts.Bastion.Key,
			User:    sshOpts.Bastion.User,
			Port:    fmt.Sprintf("%d", sshOpts.Bastion.Port),
			Timeout: connectionTimeout,
		}
	}

	if debug {
		PrettyPrintDebug("Executing command: %s via ssh %+v\n", cmd, sshConf)
	}

	output, outErr, timeout, err := sshConf.Run(cmd, commandTimeout)
	output = strings.TrimSpace(output)
	outErr = strings.TrimSpace(outErr)
	if debug {
		PrettyPrintDebug("Result of command:\nOutput: %s\nErrOutput: %s\nTimeout: %s\nErr: %s",
			output, outErr, timeout, err)
	}

	if outErr != "" {
		if err != nil {
			err = fmt.Errorf("%s %s", outErr, err)
		} else {
			err = fmt.Errorf("%s", outErr)
		}
	}

	return output, err
}

// Shell runs the command, binding Stdin, Stdout and Stderr
func shell(binaryPath string, debug bool, args ...string) (string, error) {
	cmd := exec.Command(binaryPath, args...)
	if debug {
		cmdDebug := append([]string{}, cmd.Args...)
		fmt.Printf("Executing command: %s\n", cmdDebug)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	o, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(o))
	if debug {
		fmt.Printf("Result of command:\nResult: %sErr: %s\n", output, err)
	}

	return output, err
}

func GetFirstAccessibleNode(sshOpts types.SSHConfig, nodes []types.Node, debug bool) types.Node {
	if IsNodeAddressValid(sshOpts.LocalOn) {
		for _, n := range nodes {
			if NodeEquals(sshOpts.LocalOn, n) {
				return n
			}
		}
	}

	for _, n := range nodes {
		_, err := PerformSSHCmd(sshOpts, n, "hostname", debug)
		if err == nil {
			return n
		}
	}

	return types.Node{}
}