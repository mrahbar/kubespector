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

const commandTimeout = 15

func PerformSSHCmd(sshOpts types.SSHConfig, node types.Node, cmd string, debug bool) (string, error) {
	if NodeEquals(sshOpts.LocalOn, node) {
		splits := strings.SplitN(cmd, " ", 1)
		return shell(splits[0], debug, splits[1])
	}

	nodeAddress := GetNodeAddress(node)
	key, err := validUnencryptedPrivateKey(sshOpts.Connection.Key, debug)
	if err != nil {
		if debug {
			PrettyPrintDebug("Error validating private key: %s", err)
		}
		return "", err
	}

	t := connectionTimeout(sshOpts.Connection)

	sshConf := &easyssh.MakeConfig{
		User:    sshOpts.Connection.User,
		Port:    fmt.Sprintf("%d", sshOpts.Connection.Port),
		KeyPath: key,
		Server:  nodeAddress,
		Timeout: t,
	}

	if IsNodeAddressValid(sshOpts.Bastion.Node) {
		sshConf.Proxy = easyssh.DefaultConfig{
			Server:  GetNodeAddress(sshOpts.Bastion.Node),
			KeyPath: sshOpts.Bastion.Key,
			User:    sshOpts.Bastion.User,
			Port:    fmt.Sprintf("%d", sshOpts.Bastion.Port),
			Timeout: t,
		}
	}

	if debug {
		PrettyPrintDebug("Executing command: %s via ssh %+v\n", cmd, sshConf)
	}

	output, outErr, timeout, err := sshConf.Run(cmd, commandTimeout)
	output = strings.TrimSpace(output)
	outErr = strings.TrimSpace(outErr)
	if debug {
		errFormatted := ""
		if err != nil {
			errFormatted = fmt.Sprintf("%s", err)
		}
		PrettyPrintDebug("Result of command:\nOutput: %s\nErrOutput: %s\nTimeout: %t\nErr: %s\n",
			output, outErr, timeout, errFormatted)
	}

	if outErr != "" && err != nil {
		err = fmt.Errorf("%s %s", outErr, err)
	}

	return output, err
}

func connectionTimeout(conn types.SSHConnection) time.Duration {
	t := 10 * time.Second
	if conn.Timeout > 0 {
		t = time.Duration(conn.Timeout) * time.Second
	}

	return t
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