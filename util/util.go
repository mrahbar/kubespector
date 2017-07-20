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
