package cmd

import (
	"github.com/spf13/cobra"
)

type EtcdCliOpts struct {
	clientCertAuth bool
	endpoint       string
	caFile         string
	clientCertFile string
	clientKeyFile  string
}

// etcdCmd represents the etcd command
var EtcdCmd = &cobra.Command{
	Use:   "etcd",
	Short: "Executes various actions on a etcd cluster",
	Long:  `Root command to call various actions on a etcd cluster. Please use actual subcommands.`,
}

func init() {
	RootCmd.AddCommand(EtcdCmd)
}
