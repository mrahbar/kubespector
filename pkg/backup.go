package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const localBackupDir = "/tmp"
const localEtcdBackupDir = "/tmp/etcd-backup"

var etcdBackupOpts *types.EtcdBackupOpts
var archiveName string

func Backup(cmdParams *types.CommandContext) {
    initParams(cmdParams)
    etcdBackupOpts = cmdParams.Opts.(*types.EtcdBackupOpts)

    group := util.FindGroupByName(cmdParams.Config.ClusterGroups, types.ETCD_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
        cmdParams.Printer.PrintCritical("No host configured for group [%s]", types.ETCD_GROUPNAME)
	}

    node := ssh.GetFirstAccessibleNode(config.Ssh.LocalOn, cmdExecutor, group.Nodes)

	if !util.IsNodeAddressValid(node) {
        printer.PrintCritical("No node available for etcd backup")
	}

	cmdExecutor.SetNode(node)
    printer.PrintNewLine()
	initializeOutputFile()
	backup()
	transferBackup()
}

func initializeOutputFile() {
	archiveName = fmt.Sprintf("etcd-backup-%s.tar.gz", strings.Replace(time.Now().Format("2006-01-02T15:04:05"), ":", "-", -1))

	if etcdBackupOpts.Output == "" {
		ex, err := util.GetExecutablePath()
		if err != nil {
			printer.PrintCritical("Could not get current executable path: %s", err)
		}

		etcdBackupOpts.Output = filepath.Join(ex, archiveName)
	} else {
		etcdBackupOpts.Output = filepath.Join(etcdBackupOpts.Output, archiveName)
	}
}

func backup() {
	etcdConnection := fmt.Sprintf("--endpoint='%s'", etcdBackupOpts.Endpoint)

	if etcdBackupOpts.ClientCertAuth {
		etcdConnection = fmt.Sprintf("%s --cert-file=%s --key-file=%s --ca-file=%s",
			etcdConnection, etcdBackupOpts.ClientCertFile, etcdBackupOpts.ClientKeyFile, etcdBackupOpts.CaFile)
	}

    printer.PrintInfo("Start backup process")
	cmdExecutor.DeleteRemoteFile(localEtcdBackupDir)
	backupCmd := fmt.Sprintf("etcdctl %s backup --data-dir %s --backup-dir %s", etcdConnection, etcdBackupOpts.DataDir, localEtcdBackupDir)
	_, err := cmdExecutor.PerformCmd(backupCmd, etcdBackupOpts.Sudo )

	if err != nil {
        printer.PrintCritical("Error trying to backup etcd: %s", err)
	} else {
		chmodCmd := fmt.Sprintf("chmod -R 777 %s", localEtcdBackupDir)
		cmdExecutor.PerformCmd(chmodCmd, etcdBackupOpts.Sudo )
        printer.PrintOk("Backup created")
    }

    printer.PrintNewLine()
}

func transferBackup() {
    printer.PrintInfo("Creating archive of etcd backup")
	backupArchive := path.Join(localBackupDir, archiveName)

	archiveCmd := fmt.Sprintf("tar -czvf %s -C %s .", backupArchive, localEtcdBackupDir)
    _, err := cmdExecutor.PerformCmd(archiveCmd, etcdBackupOpts.Sudo)

	if err != nil {
        printer.PrintCritical("Error trying to archive backup etcd: %s", err)
	} else {
        cmdExecutor.DeleteRemoteFile(localEtcdBackupDir)

        printer.PrintInfo("Transferring archive")
        printer.PrintNewLine()

        err = cmdExecutor.DownloadFile(backupArchive, etcdBackupOpts.Output)
        cmdExecutor.DeleteRemoteFile(backupArchive)

		if err != nil {
            printer.PrintCritical("Error trying transfer backup archive: %s", err)
		} else {
            printer.PrintOk("Etcd backup is at %s", etcdBackupOpts.Output)
		}
	}
}
