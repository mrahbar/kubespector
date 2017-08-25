package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"regexp"
	"os"
	"path/filepath"
	"path"
)

const (
	dirType = "dir"
	fileType = "file"
	noneType = "none"
)

var scpOpts *types.ScpOpts

func Scp(config types.Config, opts *types.ScpOpts) {
	scpOpts = opts
	runGeneric(config, &opts.GenericOpts, initializeScp, scp)
}

func initializeScp(target string, node string, group string) {
	if !regexp.MustCompile("^(up|u|down|d){1}$").MatchString(target) {
		util.PrettyPrintErr("Direction must either be 'up' or 'down' resp. first letter. Provided: '%s'", target)
		os.Exit(1)
	}

	if group != "" {
		util.PrintHeader(fmt.Sprintf("Executing scp in group [%s] ", group), '=')
	}

	if node != "" {
		util.PrintHeader(fmt.Sprintf("Executing scp on node %s :\n", node), '=')
	}

	util.PrettyNewLine()
}

func scp(sshOpts types.SSHConfig, target string, node types.Node, debug bool) {
	var scpErr error
	direction := ""

	localStat, err := os.Stat(scpOpts.LocalPath)
	if err != nil {
		util.PrettyPrintErr("Error downloading %s to %s: %s", scpOpts.RemotePath, scpOpts.LocalPath, err)
		os.Exit(1)
	}

	localPathIsDir := false
	if m := localStat.Mode(); m.IsDir() {
		localPathIsDir = true
	}

	if regexp.MustCompile("^(up|u){1}$").MatchString(target) {
		direction = "->"

		remoteType, err := typeOfRemotePath(sshOpts, node, scpOpts.RemotePath, debug)

		if err != nil {
			util.PrettyPrintErr("Remote path %s is unprocessable: %s", scpOpts.RemotePath, err)
		} else {
			if localPathIsDir {
				if fileType == remoteType {
					util.PrettyPrintErr("Can not upload directory %s to remote file %s. Please choose a remote directory", scpOpts.LocalPath, scpOpts.RemotePath)
					os.Exit(1)
				}

				scpErr = ssh.UploadDirectory(sshOpts, node, scpOpts.RemotePath, scpOpts.LocalPath, scpOpts.Debug)
			} else {
				if fileType == remoteType {
					util.PrettyPrintErr("Can not upload local file %s to existing remote file %s. Please choose a remote directory or a new remote filename", scpOpts.LocalPath, scpOpts.RemotePath)
					os.Exit(1)
				} else if dirType == remoteType {
					scpOpts.RemotePath = path.Join(scpOpts.RemotePath, filepath.Base(scpOpts.LocalPath))
				} //noneType means remote file name was specified but file does not exists, ssh.UploadFile will create it

				scpErr = ssh.UploadFile(sshOpts, node, scpOpts.RemotePath, scpOpts.LocalPath, scpOpts.Debug)
			}
		}
	} else if regexp.MustCompile("^(down|d){1}$").MatchString(target) {
		direction = "<-"
		remoteType, err := typeOfRemotePath(sshOpts, node, scpOpts.RemotePath, debug)

		if err != nil {
			util.PrettyPrintErr("Remote path %s is unprocessable: %s", scpOpts.RemotePath, err)
		} else {
			if dirType == remoteType {
				if !localPathIsDir {
					util.PrettyPrintErr("Can not download remote folder %s to local file %s", scpOpts.RemotePath, scpOpts.LocalPath)
					os.Exit(1)
				}

				scpErr = ssh.DownloadDirectory(sshOpts, node, scpOpts.RemotePath, scpOpts.LocalPath, scpOpts.Debug)
			} else if fileType == remoteType {
				if localPathIsDir {
					scpOpts.LocalPath = filepath.Join(scpOpts.LocalPath, filepath.Base(scpOpts.RemotePath))
				} else {
					util.PrettyPrintErr("Can not download remote file %s to existing local file %s", scpOpts.RemotePath, scpOpts.LocalPath)
					os.Exit(1)
				}

				scpErr = ssh.DownloadFile(sshOpts, node, scpOpts.RemotePath, scpOpts.LocalPath, scpOpts.Debug)
			}
		}
	}

	util.PrettyPrint(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))
	if scpErr != nil {
		util.PrettyPrintErr("Scp failed %s %s %s: %s", scpOpts.LocalPath, direction, scpOpts.RemotePath, scpErr)
	} else if direction != "" {
		util.PrettyPrintOk("Scp %s %s %s finished", scpOpts.LocalPath, direction, scpOpts.RemotePath)
	}

	util.PrettyNewLine()
}

func typeOfRemotePath(sshOpts types.SSHConfig, node types.Node, path string, debug bool) (string, error) {
	command := fmt.Sprintf(`if [ -d %s ] ; then echo "%s" ; elif [ -f %s ] ; then echo "%s"; else echo "%s"; fi;`, path, dirType, path, fileType, noneType)
	sshOut, err := ssh.PerformCmd(sshOpts, node, command, debug)

	if err != nil {
		return "", err
	}

	return sshOut.Stdout, nil
}
