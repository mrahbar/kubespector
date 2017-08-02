package pkg

import (
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/types"
	"os"
	"strings"
)

func runGeneric(config types.Config, opts *types.GenericOpts, initializer types.Initializer, processor types.Processor) {
	if opts.NodeArg != "" {
		node := types.Node{}

		for _, group := range config.ClusterGroups {
			for _, n := range group.Nodes {
				if n.Host == opts.NodeArg || n.IP == opts.NodeArg {
					node = n
					break
				}
			}
		}

		if !integration.IsNodeAddressValid(node) {
			integration.PrettyPrintErr("No node found for %v in config", opts.NodeArg)
			os.Exit(1)
		} else {
			initializer(opts.TargetArg, integration.ToNodeLabel(node), "")
			processor(config.Ssh, opts.TargetArg, node, opts.Debug)
		}
	} else {
		var groups = []string{}
		var nodes = []string{}

		if opts.GroupArg != "" {
			groups = strings.Split(opts.GroupArg, ",")
		} else {
			for _, group := range config.ClusterGroups {
				groups = append(groups, group.Name)
			}
		}

		for _, element := range groups {
			group := integration.FindGroupByName(config.ClusterGroups, element)

			if group.Nodes != nil {
				initializer(opts.TargetArg, integration.ToNodeLabel(types.Node{}), element)
				for _, node := range group.Nodes {
					if !integration.IsNodeAddressValid(node) {
						integration.PrettyPrintErr("Current node %q has no valid address", node)
						continue
					} else {
						if !integration.ElementInArray(nodes, node.Host) {
							processor(config.Ssh, opts.TargetArg, node, opts.Debug)
							nodes = append(nodes, node.Host)
						}
					}
				}
			} else {
				integration.PrettyPrintErr("No Nodes found for group: %s", element)
			}
		}
	}
}
