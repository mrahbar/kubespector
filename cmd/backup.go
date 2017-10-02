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
	backupCmd.Flags().StringVar(&etcdBackupOpts.DataDir, "data-dir", ".", "Working directory of the etcd cluster")
	backupCmd.Flags().StringVar(&etcdBackupOpts.Endpoint, "endpoint",  "http://127.0.0.1:2379", "The URL of the etcd to use")
	backupCmd.Flags().BoolVar(&etcdBackupOpts.ClientCertAuth, "secure", false, "Secure etcd communication")
	backupCmd.Flags().StringVar(&etcdBackupOpts.ClientCertFile, "client-cert",  "", "path to client certificate")
	backupCmd.Flags().StringVar(&etcdBackupOpts.ClientKeyFile, "client-cert-key",  "", "path to client certificate key")
	backupCmd.Flags().StringVar(&etcdBackupOpts.CaFile, "ca-cert", "", "path to certificate authority")
	backupCmd.Flags().BoolVarP(&etcdBackupOpts.Sudo, "sudo", "s", false, "Run commands as sudo")
	backupCmd.MarkFlagRequired("endpoint")
	backupCmd.MarkFlagRequired("data-dir")
}

func backupRun(_ *cobra.Command, _ []string) {
    pkg.Backup(createCommandContext(etcdBackupOpts))
}
