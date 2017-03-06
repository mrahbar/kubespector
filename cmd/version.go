package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)


// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the current version of kubernetes-inspector",
	Long: `The version is aligned with the SemVer specification, e.q. 1.0.0`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(RootCmd.Use + " version " + VERSION)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
