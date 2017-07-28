package cmd

import (
	"os"
	"strings"

	"github.com/mrahbar/kubernetes-inspector/integration"

	"github.com/spf13/viper"
)

type CliOpts struct {
	groupArg  string
	nodeArg   string
	targetArg string
}

type Initializer func(target string, node string, selectedGroup string)
type Processor func(integration.SSHConfig, string, integration.Node)

func Run(opts *CliOpts, initializer Initializer, processor Processor) {
	var config integration.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		integration.PrettyPrintErr(out, "Unable to decode config: %v", err)
		os.Exit(1)
	} else {
		if opts.nodeArg != "" {
			node := integration.Node{}

			for _, group := range config.ClusterGroups {
				for _, n := range group.Nodes {
					if n.Host == opts.nodeArg || n.IP == opts.nodeArg {
						node = n
						break
					}
				}
			}

			if !integration.IsNodeAddressValid(node) {
				integration.PrettyPrintErr(out, "No node found for %v in config", opts.nodeArg)
				os.Exit(1)
			} else {
				initializer(opts.targetArg, integration.ToNodeLabel(node), "")
				processor(config.Ssh, opts.targetArg, node)
			}
		} else {
			var groups = []string{}
			var nodes = []string{}

			if opts.groupArg != "" {
				groups = strings.Split(opts.groupArg, ",")
			} else {
				for _, group := range config.ClusterGroups {
					groups = append(groups, group.Name)
				}
			}

			for _, element := range groups {
				group := integration.FindGroupByName(config.ClusterGroups, element)

				if group.Nodes != nil {
					initializer(opts.targetArg, integration.ToNodeLabel(integration.Node{}), element)
					for _, node := range group.Nodes {
						if !integration.IsNodeAddressValid(node) {
							integration.PrettyPrintErr(out, "Current node %q has no valid address", node)
							continue
						} else {
							if !integration.ElementInArray(nodes, node.Host) {
								processor(config.Ssh, opts.targetArg, node)
								nodes = append(nodes, node.Host)
							}
						}
					}
				} else {
					integration.PrettyPrintErr(out, "No Nodes found for group: %s", element)
				}
			}
		}
	}
}
