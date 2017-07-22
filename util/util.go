package util

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"strings"
)

func IsNodeAddressValid(node integration.Node) bool {
	if node.Host == "" && node.IP == "" {
		return false
	} else {
		return true
	}
}

func ToNodeLabel(node integration.Node) string {
	if !IsNodeAddressValid(node) {
		return ""
	}

	label := fmt.Sprintf("%s", node.Host)

	if node.IP != "" {
		label = fmt.Sprintf("%s (%s)", label, node.IP)
	}

	return label
}

func FindGroupByName(clustergroups []integration.ClusterGroup, name string) integration.ClusterGroup {
	for _, group := range clustergroups {
		if strings.EqualFold(group.Name, name) {
			return group
		}
	}

	return integration.ClusterGroup{}
}

func ElementInArray(array []string, element string) bool {
	contains := false
	for _, v := range array {
		if v == element {
			contains = true
			break
		}
	}

	return contains
}

func RetrieveKubectlNode(nodes []integration.Node, debug bool) integration.Node {
	var node integration.Node

	for _, n := range nodes {
		nodeAddress := n.IP
		if nodeAddress == "" {
			nodeAddress = n.Host
		}

		result, err := integration.Ping(nodeAddress, n.Host)
		if debug {
			fmt.Printf("Result for ping on %s:\n\tResult: %s\tErr: %s\n", n.Host, result, err)
		}
		if err == nil {
			node = n
			break
		}
	}

	return node
}