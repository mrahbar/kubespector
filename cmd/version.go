package cmd

import (
	"runtime"

    "github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the current version and build date",
	Long:  `The version is aligned with the SemVer specification, e.q. 1.0.0`,
	Run: func(cmd *cobra.Command, args []string) {
        integration.PrettyPrint("kubernetes-inspector:")
        integration.PrettyPrint("-  Version: %s", Version)
        integration.PrettyPrint("-  Built: %s", BuildDate)
        integration.PrettyPrint("-  Go Version: %s", runtime.Version())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
