package ssh

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"io"
	"os"
)

func DownloadFile(sshOpts types.SSHConfig, node types.Node, remotePath string, localPath string, debug bool) error {
	nodeAddress := util.GetNodeAddress(node)

	if debug {
		util.PrettyPrintDebug("Copying from remote file %s:%s to %s", nodeAddress, remotePath, localPath)
	}

	if util.NodeEquals(sshOpts.LocalOn, node) {
		return copyFile(remotePath, localPath)
	}

	comm, err := establishSSHCommunication(sshOpts, util.GetNodeAddress(node), debug)
	if err != nil {
		if debug {
			util.PrettyPrintDebug("Creating communicator failed: %s", err)
		}
		return err
	}

	dstFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	err = comm.Download(remotePath, dstFile)
	if debug {
		errFormatted := "no-error"
		if err != nil {
			errFormatted = fmt.Sprintf("%s", err)
		}
		util.PrettyPrintDebug("Result of scp from remote: %s", errFormatted)
	}

	return err
}

func DownloadDirectory(sshOpts types.SSHConfig, node types.Node, remotePath string, localPath string, debug bool) error {
	nodeAddress := util.GetNodeAddress(node)

	if debug {
		util.PrettyPrintDebug("Copying from remote file %s:%s to %s", nodeAddress, remotePath, localPath)
	}

	if util.NodeEquals(sshOpts.LocalOn, node) {
		return fmt.Errorf("Local scp ist not supported")
	}

	comm, err := establishSSHCommunication(sshOpts, util.GetNodeAddress(node), debug)
	if err != nil {
		if debug {
			util.PrettyPrintDebug("Creating communicator failed: %s", err)
		}
		return err
	}

	err = comm.DownloadDir(remotePath, localPath, []string{})
	if debug {
		errFormatted := "no-error"
		if err != nil {
			errFormatted = fmt.Sprintf("%s", err)
		}
		util.PrettyPrintDebug("Result of scp from remote: %s", errFormatted)
	}

	return err
}

func UploadFile(sshOpts types.SSHConfig, node types.Node, remotePath string, localPath string, debug bool) error {
	nodeAddress := util.GetNodeAddress(node)

	if debug {
		util.PrettyPrintDebug("Copying file %s to remote %s:%s", localPath, nodeAddress, remotePath)
	}

	if util.NodeEquals(sshOpts.LocalOn, node) {
		return copyFile(localPath, remotePath)
	}

	srcFile, err := os.Open(localPath)
	defer srcFile.Close()
	if err != nil {
		return err
	}

	fi, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	comm, err := establishSSHCommunication(sshOpts, util.GetNodeAddress(node), debug)
	if err != nil {
		if debug {
			util.PrettyPrintDebug("Creating communicator failed: %s", err)
		}
		return err
	}

	err = comm.Upload(remotePath, srcFile, &fi)
	if debug {
		errFormatted := "no-error"
		if err != nil {
			errFormatted = fmt.Sprintf("%s", err)
		}
		util.PrettyPrintDebug("Result of scp from remote: %s", errFormatted)
	}

	return err
}

func UploadDirectory(sshOpts types.SSHConfig, node types.Node, remotePath string, localPath string, debug bool) error {
	nodeAddress := util.GetNodeAddress(node)

	if debug {
		util.PrettyPrintDebug("Copying directory %s to remote %s:%s", localPath, nodeAddress, remotePath)
	}

	if util.NodeEquals(sshOpts.LocalOn, node) {
		return fmt.Errorf("Local scp ist not supported")
	}

	comm, err := establishSSHCommunication(sshOpts, util.GetNodeAddress(node), debug)
	if err != nil {
		if debug {
			util.PrettyPrintDebug("Creating communicator failed: %s", err)
		}
		return err
	}

	err = comm.UploadDir(remotePath, localPath, []string{})
	if debug {
		errFormatted := "no-error"
		if err != nil {
			errFormatted = fmt.Sprintf("%s", err)
		}
		util.PrettyPrintDebug("Result of scp from remote: %s", errFormatted)
	}

	return err
}

func DeleteRemoteFile(sshOpts types.SSHConfig, node types.Node, remoteFile string, debug bool) error {
	_, err := PerformCmd(sshOpts, node, fmt.Sprintf("rm -f %s", remoteFile), debug)
	if err != nil {
		_, err2 := PerformCmd(sshOpts, node, fmt.Sprintf("sudo rm -f %s", remoteFile), debug)
		if err2 != nil {
			return flattenMultiError([]error{err, err2})
		} else {
			return err
		}
	} else {
		return nil
	}
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
