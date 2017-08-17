package integration

import (
	"fmt"
	"os"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/appleboy/easyssh-proxy"
	"io"
	"strings"
)

func PerformSCPCmdFromRemote(sshOpts types.SSHConfig, node types.Node, remotePath string, localPath string, debug bool) error {
	nodeAddress := GetNodeAddress(node)

	if debug {
		PrettyPrintDebug("Copying from remote file %s:%s to %s", nodeAddress, remotePath, localPath)
	}

	if NodeEquals(sshOpts.LocalOn, node) {
		return copyFile(remotePath, localPath)
	}

	key, err := validUnencryptedPrivateKey(sshOpts.Connection.Key, debug)
	if err != nil {
		return err
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
		PrettyPrintDebug("Executing scp %+v\n", sshConf)
	}

	output, outErr, timeout, err := sshConf.Run(fmt.Sprintf("cat %s", remotePath), commandTimeout)
	output = strings.TrimSpace(output)
	outErr = strings.TrimSpace(outErr)
	if debug {
		PrettyPrintDebug("Result of command:\nErrOutput: %s\nTimeout: %s\nErr: %s", outErr, timeout, err)
	}

	if err != nil {
		return err
	}

	fileHandler, srcErr := os.Open(localPath)
	if srcErr != nil {
		return srcErr
	}

	_, err = fileHandler.WriteString(output)

	return err
}

func PerformSCPCmdToRemote(sshOpts types.SSHConfig, node types.Node, remotePath string, localPath string, debug bool) error {
	nodeAddress := GetNodeAddress(node)

	if debug {
		PrettyPrintDebug("Copying file %s to remote %s:%s", localPath, nodeAddress, remotePath)
	}

	if NodeEquals(sshOpts.LocalOn, node) {
		return copyFile(localPath, remotePath)
	}

	key, err := validUnencryptedPrivateKey(sshOpts.Connection.Key, debug)
	if err != nil {
		return err
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
		PrettyPrintDebug("Executing scp %+v\n", sshConf)
	}

	return sshConf.Scp(localPath, remotePath)
}

func copyFile(src, dst string) error {
	dstFile, err := os.Open(dst)
	defer dstFile.Close()
	if err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	defer srcFile.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(dstFile, srcFile)
	return err
}
