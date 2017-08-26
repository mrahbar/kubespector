package cmd

import (
	"os"

    "github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Version string
var BuildDate string

var logLevelRaw string
var debug bool
var configFile string

var printer *integration.Printer

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
    RootCmd.PersistentFlags().StringVarP(&configFile, "config", "f", "./kubespector.yaml", "Path to config file")
    RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Set log-level to DEBUG")
    RootCmd.PersistentFlags().StringVar(&logLevelRaw, "log-level", "INFO", "Logging level, valid values: CRITICAL,ERROR,WARNING,INFO,DEBUG,TRACE")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
    if configFile != "" { // enable ability to specify config file via flag
        viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("kubespector") // name of config file (without extension)
	}

	viper.AddConfigPath("$HOME") // adding home directory as first search path
    viper.AddConfigPath(".")     // adding current directory as second search path
	viper.AutomaticEnv()         // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
        integration.PrettyPrint("Loading config file: %s", viper.ConfigFileUsed())
	} else {
        integration.PrettyPrintErr("Error loading config file: %s", err.Error())
        os.Exit(-1)
    }

    setLogLevel()
}

func setLogLevel() {
    ll, err := integration.ParseLogLevel(logLevelRaw)
    if err != nil {
        integration.PrettyPrintWarn("Failed to set log level %s fallback to INFO. %+v", logLevelRaw, err)
    }

    if debug && ll < integration.DEBUG {
        ll = integration.DEBUG
    }

    printer = &integration.Printer{
        LogLevel: ll,
	}
}
