package pkg

import (
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
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

		if !util.IsNodeAddressValid(node) {
			util.PrettyPrintErr("No node found for %v in config", opts.NodeArg)
			os.Exit(1)
		} else {
			initializer(opts.TargetArg, util.ToNodeLabel(node), "")
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
			group := util.FindGroupByName(config.ClusterGroups, element)

			if group.Nodes != nil {
				initializer(opts.TargetArg, util.ToNodeLabel(types.Node{}), element)
				for _, node := range group.Nodes {
					if !util.IsNodeAddressValid(node) {
						util.PrettyPrintErr("Current node %q has no valid address", node)
						continue
					} else {
						if !util.ElementInArray(nodes, node.Host) {
							processor(config.Ssh, opts.TargetArg, node, opts.Debug)
							nodes = append(nodes, node.Host)
						}
					}
				}
			} else {
				util.PrettyPrintErr("No Nodes found for group: %s", element)
			}
		}
	}
}
