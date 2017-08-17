package cmd

import (
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

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "kubespector",
	Short: "Management tool for Kubernetes",
	Long:  `Kubespector can perform various actions on a Kubernetes cluster via ssh.`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string, buildDate string) {
	Version = version
	BuildDate = buildDate
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVarP(&RootOpts.ConfigFile, "config", "f", "./kubespector.yaml", "Path to config file")
	RootCmd.PersistentFlags().BoolVarP(&RootOpts.Debug, "debug", "d", false, "Enable debug")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if RootOpts.ConfigFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(RootOpts.ConfigFile)
	} else {
		viper.SetConfigName("kubespector") // name of config file (without extension)
	}

	viper.AddConfigPath("$HOME") // adding home directory as first search path
	viper.AddConfigPath(".")     // adding home directory as first search path
	viper.AutomaticEnv()         // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		integration.PrettyPrint("Loading config file: %s", viper.ConfigFileUsed())
	} else {
		integration.PrettyPrintErr("Error loading config file: %s", err.Error())
	}
}
