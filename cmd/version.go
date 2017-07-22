package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the current version and build date",
	Long:  `The version is aligned with the SemVer specification, e.q. 1.0.0`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(out, "kubernetes-inspector:")
		fmt.Fprintf(out, "  Version: %s\n", Version)
		fmt.Fprintf(out, "  Built: %s\n", BuildDate)
		fmt.Fprintf(out, "  Go Version: %s\n", runtime.Version())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
