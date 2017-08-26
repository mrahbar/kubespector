package pkg

import (
	"fmt"
    "github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const localBackupDir = "/tmp"
const localEtcdBackupDir = "/tmp/etcd-backup"

var etcdBackupOpts *types.EtcdBackupOpts
var archiveName string

func Backup(cmdParams *types.CommandParams) {
    initParams(cmdParams)
    etcdBackupOpts = cmdParams.Opts.(*types.EtcdBackupOpts)

    group := util.FindGroupByName(cmdParams.Config.ClusterGroups, types.ETCD_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
        cmdParams.Printer.PrintErr("No host configured for group [%s]", types.ETCD_GROUPNAME)
		os.Exit(1)
	}

    node := ssh.GetFirstAccessibleNode(cmdParams.Config.Ssh, group.Nodes, printer)

    cmdExecutor := &ssh.CommandExecutor{
        SshOpts: cmdParams.Config.Ssh,
        Printer: printer,
        Node:    node,
    }

	if !util.IsNodeAddressValid(node) {
        printer.PrintErr("No node available for etcd backup")
		os.Exit(1)
	}

    integration.PrettyNewLine()
	initializeOutputFile()
    backup(cmdExecutor)
    transferBackup(cmdExecutor)
}

func initializeOutputFile() {
	archiveName = fmt.Sprintf("etcd-backup-%s.tar.gz", strings.Replace(time.Now().Format("2006-01-02T15:04:05"), ":", "-", -1))

	if etcdBackupOpts.Output == "" {
		ex, err := util.GetExecutablePath()
		if err != nil {
			os.Exit(1)
		}

		etcdBackupOpts.Output = filepath.Join(ex, archiveName)
	} else {
		etcdBackupOpts.Output = filepath.Join(etcdBackupOpts.Output, archiveName)
	}
}

func backup(cmdExecutor *ssh.CommandExecutor) {
	etcdConnection := fmt.Sprintf("--endpoint='%s'", etcdBackupOpts.Endpoint)

	if etcdBackupOpts.ClientCertAuth {
		etcdConnection = fmt.Sprintf("%s --cert-file=%s --key-file=%s --ca-file=%s",
			etcdConnection, etcdBackupOpts.ClientCertFile, etcdBackupOpts.ClientKeyFile, etcdBackupOpts.CaFile)
	}

    printer.PrintInfo("Start backup process")
    cmdExecutor.DeleteRemoteFile(localEtcdBackupDir)
	backupCmd := fmt.Sprintf("sudo etcdctl %s backup --data-dir %s --backup-dir %s", etcdConnection, etcdBackupOpts.DataDir, localEtcdBackupDir)
    _, err := cmdExecutor.PerformCmd(backupCmd)

	if err != nil {
        printer.PrintErr("Error trying to backup etcd: %s", err)
		os.Exit(1)
	} else {
        cmdExecutor.PerformCmd(fmt.Sprintf("sudo chmod -R 777 %s", localEtcdBackupDir))
        printer.PrintOk("Backup created")
    }

    integration.PrettyNewLine()
}

func transferBackup(cmdExecutor *ssh.CommandExecutor) {
    printer.PrintInfo("Creating archive of etcd backup")
	backupArchive := path.Join(localBackupDir, archiveName)

	archiveCmd := fmt.Sprintf("tar -czvf %s -C %s .", backupArchive, localEtcdBackupDir)
    _, err := cmdExecutor.PerformCmd(archiveCmd)

	if err != nil {
        printer.PrintErr("Error trying to archive backup etcd: %s", err)
		os.Exit(1)
	} else {
        cmdExecutor.DeleteRemoteFile(localEtcdBackupDir)

        printer.PrintInfo("Transferring archive")
        integration.PrettyNewLine()

        err = cmdExecutor.DownloadFile(backupArchive, etcdBackupOpts.Output)
        cmdExecutor.DeleteRemoteFile(backupArchive)

		if err != nil {
            printer.PrintErr("Error trying transfer backup archive: %s", err)
			os.Exit(1)
		} else {
            printer.PrintOk("Etcd backup is at %s", etcdBackupOpts.Output)
		}
	}
}
