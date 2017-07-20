package cmd

import (
	"io"
	"os"

	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type rootCliOpts struct {
	ConfigFile string
	Debug      bool
}

var Version string
var BuildDate string

var RootOpts = &rootCliOpts{}
var out io.Writer = os.Stdout

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "kubernetes-inspector",
	Short: "Management of Kubernetes services",
	Long: `Kubernetes-Inspector will examine the status of the different Kubernetes services running on
	specified hosts e.g. Master, Etcd, Worker, Ingress via ssh. It also provides the option to restart
	services if the ssh-user has the corresponding privileges.`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string, buildDate string) {
	Version = version
	BuildDate = buildDate
	if err := RootCmd.Execute(); err != nil {
		integration.PrettyPrintErr(out, "Error starting kubernetes-inspector: %s", err.Error())
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&RootOpts.ConfigFile, "config", "", "config file (default is ./kubernetes-inspector.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.PersistentFlags().BoolVarP(&RootOpts.Debug, "debug", "d", false, "Enable debug")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if RootOpts.ConfigFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(RootOpts.ConfigFile)
	} else {
		viper.SetConfigName("kubernetes-inspector") // name of config file (without extension)
	}

	viper.AddConfigPath("$HOME") // adding home directory as first search path
	viper.AddConfigPath(".")     // adding home directory as first search path
	viper.AutomaticEnv()         // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		integration.PrettyPrint(out, "Loading config file: %s\n", viper.ConfigFileUsed())
	} else {
		integration.PrettyPrintErr(out, "Error loading config file: %s", err.Error())
	}
}
