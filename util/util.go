package util

import "github.com/mrahbar/kubernetes-inspector/integration"

func IsNodeAddressValid(node integration.Node) bool {
	if node.Host == "" && node.IP == "" {
		return false
	} else {
		return true
	}
}
