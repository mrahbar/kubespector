package cmd

import (
	"github.com/spf13/cobra"
)

// serviceCmd represents the service command
var ServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Execute various actions on system services",
	Long:  `Root command to call various actions on system services. Please use actual subcommands.`,
}

func init() {
	RootCmd.AddCommand(ServiceCmd)
}
