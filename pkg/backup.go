package pkg

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/types"
	"os"
	"path"
	"strings"
	"time"
)

const localBackupDir = "/tmp"
const localEtcdBackupDir = "/tmp/etcd-backup"

var etcdBackupOpts *types.EtcdBackupOpts

func Backup(config types.Config, opts *types.EtcdBackupOpts) {
	etcdBackupOpts = opts
	group := integration.FindGroupByName(config.ClusterGroups, types.ETCD_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		integration.PrettyPrintErr("No host configured for group [%s]", types.ETCD_GROUPNAME)
		os.Exit(1)
	}

	node := integration.GetFirstAccessibleNode(sshOpts.LocalOn, group.Nodes, etcdBackupOpts.Debug)

	if !integration.IsNodeAddressValid(node) {
		integration.PrettyPrintErr("No node available for etcd backup")
		os.Exit(1)
	}

	setOutputPath()
	backup(config.Ssh, node)
	transferBackup(config.Ssh, node)

	integration.PrettyPrintOk("Finished\n")
}

func setOutputPath() {
	if etcdBackupOpts.Output == "" {
		ex, err := os.Executable()
		if err != nil {
			os.Exit(1)
		}

		etcdBackupOpts.Output = path.Dir(ex)
	}
}

func backup(ssh types.SSHConfig, node types.Node) {
	etcdConnection := fmt.Sprintf("--endpoint='%s'", etcdBackupOpts.Endpoint)

	if etcdBackupOpts.ClientCertAuth {
		etcdConnection = fmt.Sprintf("%s --cert-file=%s --key-file=%s --ca-file=%s",
			etcdConnection, etcdBackupOpts.ClientCertFile, etcdBackupOpts.ClientKeyFile, etcdBackupOpts.CaFile)
	}

	integration.PrettyPrint("Start backup process\n")
	backupCmd := fmt.Sprintf("etcdctl %s backup --data-dir %s --backup-dir %s", etcdConnection, etcdBackupOpts.DataDir, localEtcdBackupDir)
	result, err := integration.PerformSSHCmd(ssh, node, backupCmd, etcdBackupOpts.Debug)

	if err != nil {
		integration.PrettyPrintErr("Error trying to backup etcd:\n\tResult: %s\tErr: %s", result, err)
		os.Exit(1)
	} else {
		integration.PrettyPrint("Backup created\n")
	}
}

func transferBackup(ssh types.SSHConfig, node types.Node) {
	integration.PrettyPrint("Creating archive of Etcd backup\n")

	archiveName := fmt.Sprintf("etcd-backup-%s.tar.gz", strings.Replace(time.Now().Format("2006-01-02T15:04:05"), ":", "-", -1))
	backupArchive := path.Join(localBackupDir, archiveName)

	archiveCmd := fmt.Sprintf("tar -czvf %s -C %s . ", backupArchive, localEtcdBackupDir)
	result, err := integration.PerformSSHCmd(ssh, node, archiveCmd, etcdBackupOpts.Debug)

	if err != nil {
		integration.PrettyPrintErr("Error trying to archive backup etcd:\n\tResult: %s\tErr: %s", result, err)
		os.Exit(1)
	} else {
		cleanUp(ssh, node, localEtcdBackupDir)

		integration.PrettyPrint("Transferring archive\n")
		err = integration.PerformSCPCmdFromRemote(ssh, node, backupArchive, etcdBackupOpts.Output, etcdBackupOpts.Debug)

		cleanUp(ssh, node, backupArchive)

		if err != nil {
			integration.PrettyPrintErr("Error trying transfer backup archive:\n\tErr: %s", err)
			os.Exit(1)
		} else {
			integration.PrettyPrint("Etcd backup is at %s\n", path.Join(etcdBackupOpts.Output, archiveName))
		}

	}
}

func cleanUp(ssh types.SSHConfig, node types.Node, dir string) {
	integration.PerformSSHCmd(ssh, node, fmt.Sprintf("rm -rf %s", dir), etcdBackupOpts.Debug)
}
