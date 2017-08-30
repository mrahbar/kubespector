package pkg

import (
	"fmt"

	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

const (
	dirType  = "dir"
	fileType = "file"
	noneType = "none"
)

var scpOpts *types.ScpOpts

func Scp(cmdParams *types.CommandContext) {
	initParams(cmdParams)
	scpOpts = cmdParams.Opts.(*types.ScpOpts)
	runGeneric(config, &scpOpts.GenericOpts, initializeScp, scp)
}

func initializeScp(target string, node string, group string) {
	if !regexp.MustCompile("^(up|u|down|d){1}$").MatchString(target) {
		printer.PrintCritical("Direction must either be 'up' or 'down' resp. first letter. Provided: '%s'", target)
	}

	if group != "" {
		printer.PrintHeader(fmt.Sprintf("Executing scp in group [%s] ", group), '=')
	}

	if node != "" {
		printer.PrintHeader(fmt.Sprintf("Executing scp on node %s :\n", node), '=')
	}

	printer.PrintNewLine()
}

func scp(target string) {
	var scpErr error
	direction := ""

	remoteType, err := typeOfRemotePath()
	if err != nil {
		printer.PrintErr("Remote path %s is unprocessable: %s", scpOpts.RemotePath, err)
		return
	}

	localType, err := typeOfLocalPath()
	if err != nil {
		printer.PrintErr("Local path %s is unprocessable: %s", scpOpts.LocalPath, err)
		return
	}

	if regexp.MustCompile("^(up|u){1}$").MatchString(target) {
		direction = "->"

		if dirType == localType {
			if fileType == remoteType {
				printer.PrintCritical("Can not upload directory %s to remote file %s. Please choose a remote directory", scpOpts.LocalPath, scpOpts.RemotePath)
			}

			scpErr = cmdExecutor.UploadDirectory(scpOpts.RemotePath, scpOpts.LocalPath)
		} else {
			if fileType == remoteType {
				printer.PrintCritical("Can not upload local file %s to existing remote file %s. Please choose a remote directory or a new remote filename", scpOpts.LocalPath, scpOpts.RemotePath)
			} else if dirType == remoteType {
				scpOpts.RemotePath = path.Join(scpOpts.RemotePath, filepath.Base(scpOpts.LocalPath))
			} //noneType means remote file name was specified but file does not exists, ssh.UploadFile will create it

			scpErr = cmdExecutor.UploadFile(scpOpts.RemotePath, scpOpts.LocalPath)
		}
	} else if regexp.MustCompile("^(down|d){1}$").MatchString(target) {
		direction = "<-"

		if dirType == remoteType {
			if fileType == localType {
				printer.PrintCritical("Can not download remote folder %s to local file %s", scpOpts.RemotePath, scpOpts.LocalPath)
			}

			scpErr = cmdExecutor.DownloadDirectory(scpOpts.RemotePath, scpOpts.LocalPath)
		} else if fileType == remoteType {
			if dirType == localType {
				scpOpts.LocalPath = filepath.Join(scpOpts.LocalPath, filepath.Base(scpOpts.RemotePath))
			} else {
				printer.PrintCritical("Can not download remote file %s to existing local file %s", scpOpts.RemotePath, scpOpts.LocalPath)
			}

			scpErr = cmdExecutor.DownloadFile(scpOpts.RemotePath, scpOpts.LocalPath)
		}
	}

	printer.Print(fmt.Sprintf("Result on node %s:", util.ToNodeLabel(node)))
	if scpErr != nil {
		printer.PrintErr("Scp failed %s %s %s: %s", scpOpts.LocalPath, direction, scpOpts.RemotePath, scpErr)
	} else if direction != "" {
		printer.PrintOk("Scp %s %s %s finished", scpOpts.LocalPath, direction, scpOpts.RemotePath)
	}

	printer.PrintNewLine()
}

func typeOfRemotePath() (string, error) {
	command := fmt.Sprintf(`if [ -d %s ] ; then echo "%s" ; elif [ -f %s ] ; then echo "%s"; else echo "%s"; fi;`,
		scpOpts.RemotePath, dirType, scpOpts.RemotePath, fileType, noneType)
	sshOut, err := cmdExecutor.PerformCmd(command)

	if err != nil {
		return "", err
	}

	return sshOut.Stdout, nil
}

func typeOfLocalPath() (string, error) {
	localStat, err := os.Stat(scpOpts.LocalPath)
	if err != nil {
		return noneType, err
	}

	m := localStat.Mode()
	if m.IsDir() {
		return dirType, nil
	} else if m.IsRegular() {
		return fileType, nil
	} else {
		return noneType, err
	}
}
