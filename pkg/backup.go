package pkg

import (
	"fmt"
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

func Backup(config types.Config, opts *types.EtcdBackupOpts) {
	etcdBackupOpts = opts
	group := util.FindGroupByName(config.ClusterGroups, types.ETCD_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		util.PrettyPrintErr("No host configured for group [%s]", types.ETCD_GROUPNAME)
		os.Exit(1)
	}

	node := ssh.GetFirstAccessibleNode(config.Ssh, group.Nodes, etcdBackupOpts.Debug)

	if !util.IsNodeAddressValid(node) {
		util.PrettyPrintErr("No node available for etcd backup")
		os.Exit(1)
	}

	util.PrettyNewLine()
	initializeOutputFile()
	backup(config.Ssh, node)
	transferBackup(config.Ssh, node)
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

func backup(sshConf types.SSHConfig, node types.Node) {
	etcdConnection := fmt.Sprintf("--endpoint='%s'", etcdBackupOpts.Endpoint)

	if etcdBackupOpts.ClientCertAuth {
		etcdConnection = fmt.Sprintf("%s --cert-file=%s --key-file=%s --ca-file=%s",
			etcdConnection, etcdBackupOpts.ClientCertFile, etcdBackupOpts.ClientKeyFile, etcdBackupOpts.CaFile)
	}

	util.PrettyPrintInfo("Start backup process")
	cleanUp(sshConf, node, localEtcdBackupDir)
	backupCmd := fmt.Sprintf("sudo etcdctl %s backup --data-dir %s --backup-dir %s", etcdConnection, etcdBackupOpts.DataDir, localEtcdBackupDir)
	_, err := ssh.PerformCmd(sshConf, node, backupCmd, etcdBackupOpts.Debug)

	if err != nil {
		util.PrettyPrintErr("Error trying to backup etcd: %s", err)
		os.Exit(1)
	} else {
		ssh.PerformCmd(sshConf, node, fmt.Sprintf("sudo chmod -R 777 %s", localEtcdBackupDir), etcdBackupOpts.Debug)
		util.PrettyPrintOk("Backup created")
	}

	util.PrettyNewLine()
}

func transferBackup(sshConf types.SSHConfig, node types.Node) {
	util.PrettyPrintInfo("Creating archive of etcd backup")
	backupArchive := path.Join(localBackupDir, archiveName)

	archiveCmd := fmt.Sprintf("tar -czvf %s -C %s .", backupArchive, localEtcdBackupDir)
	_, err := ssh.PerformCmd(sshConf, node, archiveCmd, etcdBackupOpts.Debug)

	if err != nil {
		util.PrettyPrintErr("Error trying to archive backup etcd: %s", err)
		os.Exit(1)
	} else {
		cleanUp(sshConf, node, localEtcdBackupDir)

		util.PrettyPrintInfo("Transferring archive")
		util.PrettyNewLine()

		err = ssh.DownloadFile(sshConf, node, backupArchive, etcdBackupOpts.Output, etcdBackupOpts.Debug)
		cleanUp(sshConf, node, backupArchive)

		if err != nil {
			util.PrettyPrintErr("Error trying transfer backup archive: %s", err)
			os.Exit(1)
		} else {
			util.PrettyPrintOk("Etcd backup is at %s", etcdBackupOpts.Output)
		}
	}
}

func cleanUp(sshConf types.SSHConfig, node types.Node, dir string) {
	ssh.DeleteRemoteFile(sshConf, node, dir, etcdBackupOpts.Debug)
}
