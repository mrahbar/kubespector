package pkg

import (
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"strings"
)

type Initializer func(target string, node string, group string)
type Processor func(target string)

func runGeneric(config types.Config, opts *types.GenericOpts, initializer Initializer, processor Processor) {
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
            printer.PrintCritical("No node found for %v in config", opts.NodeArg)
		} else {
			initializer(opts.TargetArg, util.ToNodeLabel(node), "")
			cmdExecutor.SetNode(node)
			processor(opts.TargetArg)
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
                        printer.PrintErr("Current node %q has no valid address", node)
						continue
					} else {
						if !util.ElementInArray(nodes, node.Host) {
							cmdExecutor.SetNode(node)
							processor(opts.TargetArg)
							nodes = append(nodes, node.Host)
						}
					}
				}
			} else {
                printer.PrintErr("No Nodes found for group: %s", element)
			}
		}
	}
}
