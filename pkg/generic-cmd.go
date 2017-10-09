package pkg

import (
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"strings"
	"sort"
)

type Initializer func(target string, node string)
type Processor func(target string)

func runGeneric(config types.Config, opts *types.GenericOpts, initializer Initializer, processor Processor) {
	if opts.TargetArg == "" {
		printer.PrintCritical("Invalid options. Parameter missing.")
	}

	totalNodes := []types.Node{}

	if opts.NodeArg != "" {
		nodes := strings.Split(opts.NodeArg, ",")

		for _,i := range nodes {
			for _, group := range config.ClusterGroups {
				for _, n := range group.Nodes {
					if n.Host == i || n.IP == i {
						if util.IsNodeAddressValid(n) && !util.NodeInArray(totalNodes, n) {
							totalNodes = append(totalNodes, n)
						}
					}
				}
			}
		}
	} else {
		var groups = []string{}

		if strings.EqualFold(opts.GroupArg, types.ALL_GROUPNAME) {
			for _, group := range config.ClusterGroups {
				groups = append(groups, group.Name)
			}
		} else if opts.GroupArg != "" {
			groups = strings.Split(opts.GroupArg, ",")
		} else {
			printer.PrintCritical("No group specified")
		}

		for _, element := range groups {
			group := util.FindGroupByName(config.ClusterGroups, element)

			for _, n := range group.Nodes {
				if util.IsNodeAddressValid(n) && !util.NodeInArray(totalNodes, n) {
					totalNodes = append(totalNodes, n)
				}
			}
		}
	}

	if len(totalNodes) == 0 {
		printer.PrintCritical("No node in current selection")
	} else {
		sort.Slice(totalNodes, func(i, j int) bool {//TODO fix ordering
			return util.GetNodeAddress(totalNodes[i]) < util.GetNodeAddress(totalNodes[j])
		})
		for _, node := range totalNodes {// TODO maybe paralle loop http://www.golangpatterns.info/concurrency/parallel-for-loop
			initializer(opts.TargetArg, util.ToNodeLabel(node))
			cmdExecutor.SetNode(node)
			processor(opts.TargetArg)
		}
	}
}
