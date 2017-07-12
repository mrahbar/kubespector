package cmd

import (
	"os"
	"reflect"
	"strings"

	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/spf13/viper"
)

type CliOpts struct {
	groupArg  string
	nodeArg   string
	targetArg string
}

type Initializer func(target string, node integration.Node, selectedGroup string)
type Processor func(*integration.SSHConfig, string, integration.Node)

func Run(opts *CliOpts, initializer Initializer, processor Processor) {
	var config integration.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		integration.PrettyPrintErr(out, "Unable to decode config: %v", err)
		os.Exit(1)
	} else {
		if opts.targetArg == "" {
			integration.PrettyPrintErr(out, "Command has been called with insufficient parameters.")
			os.Exit(1)
		}

		if opts.nodeArg != "" {
			v := reflect.ValueOf(config.Cluster)
			node := integration.Node{}

			for i := 0; i < v.NumField(); i++ {
				nodes := v.Field(i).FieldByName("Nodes").Interface().([]integration.Node)
				for _, n := range nodes {
					if n.Host == opts.nodeArg || n.IP == opts.nodeArg {
						node = n
						break
					}
				}
			}

			if node.IP != "" {
				if !util.IsNodeAddressValid(node) {
					integration.PrettyPrintErr(out, "Node %q has no valid address", node)
					os.Exit(1)
				}
				initializer(opts.targetArg, node, "")
				processor(&config.Ssh, opts.targetArg, node)
			} else {
				integration.PrettyPrintErr(out, "No node found for %v in config", opts.nodeArg)
				os.Exit(1)
			}

		} else {
			var cmdGroups []string

			if opts.groupArg != "" {
				cmdGroups = strings.Split(opts.groupArg, ",")
			} else {
				cmdGroups = ClusterMembers[:len(ClusterMembers)-1]
			}

			var nodes []integration.Node

			for _, element := range cmdGroups {
				switch element {
				case "Etcd":
					nodes = config.Cluster.Etcd.Nodes
				case "Master":
					nodes = config.Cluster.Master.Nodes
				case "Worker":
					nodes = config.Cluster.Worker.Nodes
				case "Ingress":
					nodes = config.Cluster.Ingress.Nodes
				case "Registry":
					nodes = config.Cluster.Registry.Nodes
				}

				if nodes != nil {
					initializer(opts.targetArg, integration.Node{}, element)
					for _, node := range nodes {
						if !util.IsNodeAddressValid(node) {
							integration.PrettyPrintErr(out, "Current node %q has no valid address", node)
							break
						}
						processor(&config.Ssh, opts.targetArg, node)
					}
				} else {
					integration.PrettyPrintErr(out, "No Nodes found for group: %s", element)
				}
			}
		}
	}
}
