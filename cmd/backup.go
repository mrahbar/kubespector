package cmd

import (
	"github.com/spf13/cobra"

	"github.com/mrahbar/kubernetes-inspector/pkg"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
)

var etcdBackupOpts = &types.EtcdBackupOpts{}

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:     "backup",
	Short:   "Creates a backup of an etcd cluster",
	Long:    `Backups are created via etcdctl and stored as an tar.gz archive on the local filesystem`,
	PreRunE: util.CheckRequiredFlags,
	Run:     backupRun,
}

func init() {
	EtcdCmd.AddCommand(backupCmd)
	backupCmd.Flags().StringVarP(&etcdBackupOpts.Output, "output", "o", "", "The target directory for the resulting ZIP file of the backup")
	backupCmd.Flags().StringVarP(&etcdBackupOpts.DataDir, "data-dir", "r", ".", "Working directory of the etcd cluster")
	backupCmd.Flags().StringVarP(&etcdBackupOpts.Endpoint, "endpoint", "e", "http://127.0.0.1:2379", "The URL of the etcd to use")
	backupCmd.Flags().BoolVarP(&etcdBackupOpts.ClientCertAuth, "secure", "s", false, "Secure etcd communication")
	backupCmd.Flags().StringVarP(&etcdBackupOpts.ClientCertFile, "client-cert", "c", "", "path to client certificate")
	backupCmd.Flags().StringVarP(&etcdBackupOpts.ClientKeyFile, "client-cert-key", "k", "", "path to client certificate key")
	backupCmd.Flags().StringVarP(&etcdBackupOpts.CaFile, "ca-cert", "a", "", "path to certificate authority")
	backupCmd.MarkFlagRequired("endpoint")
	backupCmd.MarkFlagRequired("data-dir")
}

func backupRun(_ *cobra.Command, _ []string) {
    pkg.Backup(createCommandContext(etcdBackupOpts))
}
