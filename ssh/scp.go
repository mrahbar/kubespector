package ssh

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/util"
	"io"
	"os"
)

func (c *CommandExecutor) DownloadFile(remotePath string, localPath string) error {
    nodeAddress := util.GetNodeAddress(c.Node)

    c.Printer.PrintDebug("Copying from remote file %s:%s to %s", nodeAddress, remotePath, localPath)

    if util.NodeEquals(c.SshOpts.LocalOn, c.Node) {
		return copyFile(remotePath, localPath)
	}

    comm, err := establishSSHCommunication(c.SshOpts, util.GetNodeAddress(c.Node), c.Printer)
	if err != nil {
        c.Printer.PrintDebug("Creating communicator failed: %s", err)
		return err
	}

	dstFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	err = comm.Download(remotePath, dstFile)
    errFormatted := "no-error"
    if err != nil {
        errFormatted = fmt.Sprintf("%s", err)
    }
    c.Printer.PrintDebug("Result of scp from remote: %s", errFormatted)

	return err
}

func (c *CommandExecutor) DownloadDirectory(remotePath string, localPath string) error {
    nodeAddress := util.GetNodeAddress(c.Node)
    c.Printer.PrintDebug("Copying from remote file %s:%s to %s", nodeAddress, remotePath, localPath)

    if util.NodeEquals(c.SshOpts.LocalOn, c.Node) {
		return fmt.Errorf("Local scp ist not supported")
	}

    comm, err := establishSSHCommunication(c.SshOpts, util.GetNodeAddress(c.Node), c.Printer)
	if err != nil {
        c.Printer.PrintDebug("Creating communicator failed: %s", err)
		return err
	}

	err = comm.DownloadDir(remotePath, localPath, []string{})
    errFormatted := "no-error"
    if err != nil {
        errFormatted = fmt.Sprintf("%s", err)
    }
    c.Printer.PrintDebug("Result of scp from remote: %s", errFormatted)

	return err
}

func (c *CommandExecutor) UploadFile(remotePath string, localPath string) error {
    nodeAddress := util.GetNodeAddress(c.Node)
    c.Printer.PrintDebug("Copying file %s to remote %s:%s", localPath, nodeAddress, remotePath)

    if util.NodeEquals(c.SshOpts.LocalOn, c.Node) {
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

    comm, err := establishSSHCommunication(c.SshOpts, util.GetNodeAddress(c.Node), c.Printer)
	if err != nil {
        c.Printer.PrintDebug("Creating communicator failed: %s", err)
		return err
	}

	err = comm.Upload(remotePath, srcFile, &fi)
    errFormatted := "no-error"
    if err != nil {
        errFormatted = fmt.Sprintf("%s", err)
    }
    c.Printer.PrintDebug("Result of scp from remote: %s", errFormatted)

	return err
}

func (c *CommandExecutor) UploadDirectory(remotePath string, localPath string) error {
    nodeAddress := util.GetNodeAddress(c.Node)
    c.Printer.PrintDebug("Copying directory %s to remote %s:%s", localPath, nodeAddress, remotePath)

    if util.NodeEquals(c.SshOpts.LocalOn, c.Node) {
		return fmt.Errorf("Local scp ist not supported")
	}

    comm, err := establishSSHCommunication(c.SshOpts, util.GetNodeAddress(c.Node), c.Printer)
	if err != nil {
        c.Printer.PrintDebug("Creating communicator failed: %s", err)
		return err
	}

	err = comm.UploadDir(remotePath, localPath, []string{})
    errFormatted := "no-error"
    if err != nil {
        errFormatted = fmt.Sprintf("%s", err)
    }
    c.Printer.PrintDebug("Result of scp from remote: %s", errFormatted)

	return err
}

func (c *CommandExecutor) DeleteRemoteFile(remoteFile string) error {
    _, err := c.PerformCmd(fmt.Sprintf("rm -f %s", remoteFile))
	if err != nil {
        _, err2 := c.PerformCmd(fmt.Sprintf("sudo rm -f %s", remoteFile))
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
