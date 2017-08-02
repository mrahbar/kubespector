package cmd

import (
	"github.com/spf13/cobra"
)

// perfCmd represents the perf command
var PerfCmd = &cobra.Command{
	Use:   "perf",
	Short: "Executes various performance tests",
	Long:  `Root command to call various performance tests. Please use actual subcommands.`,
}

func init() {
	RootCmd.AddCommand(PerfCmd)
}
