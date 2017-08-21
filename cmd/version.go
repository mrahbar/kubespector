package cmd

import (
	"runtime"

	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the current version and build date",
	Long:  `The version is aligned with the SemVer specification, e.q. 1.0.0`,
	Run: func(cmd *cobra.Command, args []string) {
		util.PrettyPrint("kubernetes-inspector:")
		util.PrettyPrint("  Version: %s", Version)
		util.PrettyPrint("  Built: %s", BuildDate)
		util.PrettyPrint("  Go Version: %s", runtime.Version())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
