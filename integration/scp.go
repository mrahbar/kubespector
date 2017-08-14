package integration

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/types"
)

func PerformSCPCmdFromRemote2(sshOpts types.SSHConfig, node types.Node, remotePath string, localPath string, debug bool) error {

	if NodeEquals(sshOpts.LocalOn, node) {
		return copyFile(remotePath, localPath)
	}

	nodeAddress := GetNodeAddress(node)

	opts := []string{}

	if IsNodeAddressValid(sshOpts.Bastion.Node) {
		proxyJump := fmt.Sprintf("-J %s@%s:%s", sshOpts.Bastion.User, GetNodeAddress(sshOpts.Bastion.Node), sshOpts.Bastion.Port)
		opts = append(opts, proxyJump)
	}

	client, err := newSCPClient(fmt.Sprintf("%s@%s:%s", sshOpts.Connection.User, nodeAddress, remotePath),
		sshOpts.Connection.Port, sshOpts.Connection.Key, opts, debug)

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

func PerformSCPCmdToRemote2(sshOpts types.SSHConfig, node types.Node, localPath string, remotePath string, debug bool) error {

	if NodeEquals(sshOpts.LocalOn, node) {
		return copyFile(localPath, remotePath)
	}

	nodeAddress := GetNodeAddress(node)

	opts := []string{}

	if IsNodeAddressValid(sshOpts.Bastion.Node) {
		proxyJump := fmt.Sprintf("-J %s@%s:%s", sshOpts.Bastion.User, GetNodeAddress(sshOpts.Bastion.Node), sshOpts.Bastion.Port)
		opts = append(opts, proxyJump)
	}

	client, err := newSCPClient(localPath, sshOpts.Connection.Port, sshOpts.Connection.Key, opts, debug)

	if err != nil {
		msg := fmt.Sprintf("Error creating SCP client for host %s: %v", nodeAddress, err)
		PrettyPrintErr(msg)
		return err
	}

	result, err := client.Output(false, debug, fmt.Sprintf("%s@%s:%s", sshOpts.Connection.User, nodeAddress, remotePath))

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
