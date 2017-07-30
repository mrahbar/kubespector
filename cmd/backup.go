package cmd

import (
	"github.com/spf13/cobra"
	"os"
	"path"
	"fmt"
	"time"

	"github.com/spf13/viper"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"strings"
)

const localBackupDir = "/tmp"
const localEtcdBackupDir = "/tmp/etcd-backup"

type etcdBackupCliOpts struct {
	output  string
	dataDir string
	EtcdCliOpts
}

var etcdBackupOpts = &etcdBackupCliOpts{}

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:     "backup",
	Short:   "Creates a backup of an etcd cluster",
	Long:    `Backups are created via etcdctl and stored as an tar.gz archive on the local filesystem`,
	PreRunE: integration.CheckRequiredFlags,
	Run:     backupRun,
}

func init() {
	EtcdCmd.AddCommand(backupCmd)
	backupCmd.Flags().StringVarP(&etcdBackupOpts.output, "output", "o", ".", "The target directory for the resulting ZIP file of the backup")
	backupCmd.Flags().StringVarP(&etcdBackupOpts.dataDir, "data-dir", "r", ".", "Working directory of the etcd cluster")
	backupCmd.Flags().StringVarP(&etcdBackupOpts.endpoint, "endpoint", "e", "http://127.0.0.1:2379", "The URL of the etcd to use")
	backupCmd.Flags().BoolVarP(&etcdBackupOpts.clientCertAuth, "secure", "s", false, "Secure etcd communication")
	backupCmd.Flags().StringVarP(&etcdBackupOpts.clientCertFile, "client-cert", "c", "", "path to client certificate")
	backupCmd.Flags().StringVarP(&etcdBackupOpts.clientKeyFile, "client-cert-key", "k", "", "path to client certificate key")
	backupCmd.Flags().StringVarP(&etcdBackupOpts.caFile, "ca-cert", "a", "", "path to certificate authority")
	backupCmd.MarkFlagRequired("endpoint")
	backupCmd.MarkFlagRequired("data-dir")
}

func backupRun(_ *cobra.Command, _ []string) {
	var config integration.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		integration.PrettyPrintErr(out, "Unable to decode config: %v", err)
		os.Exit(1)
	}

	group := integration.FindGroupByName(config.ClusterGroups, integration.ETCD_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		integration.PrettyPrintErr(out, "No host configured for group [%s]", integration.ETCD_GROUPNAME)
		os.Exit(1)
	}

	node := integration.GetFirstAccessibleNode(group.Nodes, RootOpts.Debug)

	if !integration.IsNodeAddressValid(node) {
		integration.PrettyPrintErr(out, "No node available for etcd backup")
		os.Exit(1)
	}

	setOutputPath()
	backup(config.Ssh, node)
	transferBackup(config.Ssh, node)

	integration.PrettyPrintOk(out, "Finished\n")
}

func setOutputPath() {
	if etcdBackupOpts.output == "" {
		ex, err := os.Executable()
		if err != nil {
			os.Exit(1)
		}

		etcdBackupOpts.output = path.Dir(ex)
	}
}

func backup(ssh integration.SSHConfig, node integration.Node) {
	etcdConnection := fmt.Sprintf("--endpoint='%s'", etcdBackupOpts.endpoint)

	if etcdBackupOpts.clientCertAuth {
		etcdConnection = fmt.Sprintf("%s --cert-file=%s --key-file=%s --ca-file=%s",
			etcdConnection, etcdBackupOpts.clientCertFile, etcdBackupOpts.clientKeyFile, etcdBackupOpts.caFile)
	}

	integration.PrettyPrint(out, "Start backup process\n")
	backupCmd := fmt.Sprintf("etcdctl %s backup --data-dir %s --backup-dir %s", etcdConnection, etcdBackupOpts.dataDir, localEtcdBackupDir)
	result, err := integration.PerformSSHCmd(out, ssh, node, backupCmd, RootOpts.Debug)

	if err != nil {
		integration.PrettyPrintErr(out, "Error trying to backup etcd:\n\tResult: %s\tErr: %s", result, err)
		os.Exit(1)
	} else {
		integration.PrettyPrint(out, "Backup created\n")
	}
}

func transferBackup(ssh integration.SSHConfig, node integration.Node) {
	integration.PrettyPrint(out, "Creating archive of Etcd backup\n")

	archiveName := fmt.Sprintf("etcd-backup-%s.tar.gz", strings.Replace(time.Now().Format("2006-01-02T15:04:05"), ":", "-", -1))
	backupArchive := path.Join(localBackupDir, archiveName)

	archiveCmd := fmt.Sprintf("tar -czvf %s -C %s . ", backupArchive, localEtcdBackupDir)
	result, err := integration.PerformSSHCmd(out, ssh, node, archiveCmd, RootOpts.Debug)

	if err != nil {
		integration.PrettyPrintErr(out, "Error trying to archive backup etcd:\n\tResult: %s\tErr: %s", result, err)
		os.Exit(1)
	} else {
		cleanUp(ssh, node, localEtcdBackupDir)

		integration.PrettyPrint(out, "Transferring archive\n")
		result, err = integration.PerformSCPCmdFromRemote(out, ssh, node, backupArchive, etcdBackupOpts.output, RootOpts.Debug)

		cleanUp(ssh, node, backupArchive)

		if err != nil {
			integration.PrettyPrintErr(out, "Error trying transfer backup archive:\n\tResult: %s\tErr: %s", result, err)
			os.Exit(1)
		} else {
			integration.PrettyPrint(out, "Etcd backup is at %s\n", path.Join(etcdBackupOpts.output, archiveName))
		}

	}
}
func cleanUp(ssh integration.SSHConfig, node integration.Node, dir string) {
	integration.PerformSSHCmd(out, ssh, node, fmt.Sprintf("rm -rf %s", dir), RootOpts.Debug)
}
