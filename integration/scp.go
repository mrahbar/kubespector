package integration

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/types"
	"strings"
	"io"
	"os"
)

func PerformSCPCmdFromRemote(sshOpts types.SSHConfig, node types.Node, remotePath string, localPath string, debug bool) error {

	if node.Host != "" && sshOpts.LocalOn == node.Host {
		remoteFile, err := os.Open(remotePath)
		defer remoteFile.Close()
		if err != nil {
			return err
		}

		localFile, err := os.Open(localPath)
		defer localFile.Close()
		if err != nil {
			return err
		}

		_, err = io.Copy(localFile, remoteFile)
		return err
	}

	nodeAddress := GetNodeAddress(node)

	client, err := newSCPClient(fmt.Sprintf("%s@%s:%s", sshOpts.User, nodeAddress, remotePath),
		sshOpts.Port, sshOpts.Key, strings.FieldsFunc(sshOpts.Options, func(r rune) bool {
			return r == ' ' || r == ','
		}), debug)

	if err != nil {
		msg := fmt.Sprintf("Error creating SCP client for host %s: %v", nodeAddress, err)
		PrettyPrintErr(msg)
		return err
	}

	result, err := client.Output(false, debug, localPath)

	if err != nil {
		return fmt.Errorf("Result: %s\t%s", result, err)
	} else {
		return nil
	}
}

func PerformSCPCmdToRemote(sshOpts types.SSHConfig, node types.Node, localPath string, remotePath string, debug bool) error {

	if node.Host != "" && sshOpts.LocalOn == node.Host {
		remoteFile, err := os.Open(remotePath)
		defer remoteFile.Close()
		if err != nil {
			return err
		}

		localFile, err := os.Open(localPath)
		defer localFile.Close()
		if err != nil {
			return err
		}

		_, err = io.Copy(remoteFile, localFile)
		return err
	}

	nodeAddress := GetNodeAddress(node)

	client, err := newSCPClient(localPath, sshOpts.Port, sshOpts.Key,
		strings.FieldsFunc(sshOpts.Options, func(r rune) bool {
			return r == ' ' || r == ','
		}), debug)

	if err != nil {
		msg := fmt.Sprintf("Error creating SCP client for host %s: %v", nodeAddress, err)
		PrettyPrintErr(msg)
		return err
	}

	result, err := client.Output(false, debug, fmt.Sprintf("%s@%s:%s", sshOpts.User, nodeAddress, remotePath))

	if err != nil {
		return fmt.Errorf("Result: %s\t%s", result, err)
	} else {
		return nil
	}
}

// newSCPClient verifies ssh is available in the PATH and returns an SSH client
func newSCPClient(remoteHost string, port int, key string, options []string, debug bool) (Client, error) {
	return newClient(SecureShellBinary{binaryName: "scp", portArg: "-P"}, remoteHost, port, key, options, debug)
}
