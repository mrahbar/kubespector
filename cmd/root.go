package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
)

var VERSION = "0.1.0"
var cfgFile string
var out io.Writer = os.Stdout
var ClusterMembers = []string{"Etcd", "Master", "Worker", "Ingress", "Kubernetes"}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "kubernetes-inspector",
	Short: "Management of Kubernetes services",
	Long: `Kubernetes-Insepctor will examine the status of the different Kubernetes services running on
	specified hosts e.g. Master, Etcd, Worker, Ingress via ssh. It also provides the option to restart
	services if the ssh-user has the corresponding priviliges.`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./kubernetes-inspector.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		fmt.Printf("Set config file to: %v\n", cfgFile)
		viper.SetConfigFile(cfgFile)
	}
	viper.SetConfigName("kubernetes-inspector") // name of config file (without extension)
	viper.AddConfigPath("$HOME")  // adding home directory as first search path
	viper.AddConfigPath(".")  // adding home directory as first search path
	viper.AutomaticEnv()          // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		fmt.Println("Loading config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println("Error loading config file:", err.Error())
	}
}
