package util

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/types"
)

func GetNodeAddress(node types.Node) string {
	nodeAddress := node.IP
	if nodeAddress == "" {
		nodeAddress = node.Host
	}

	return nodeAddress
}

func IsNodeAddressValid(node types.Node) bool {
	if node.Host == "" && node.IP == "" {
		return false
	} else {
		return true
	}
}

func NodeEquals(n1, n2 types.Node) bool {
	if IsNodeAddressValid(n1) && IsNodeAddressValid(n2) {
		if n1.IP != "" && n2.IP != "" {
			return n1.IP == n2.IP
		} else if n1.Host != "" && n2.Host != "" {
			return n1.Host == n2.Host
		} else {
			return false
		}
	} else {
		return false
	}
}

func ToNodeLabel(node types.Node) string {
	if !IsNodeAddressValid(node) {
		return ""
	}

	label := fmt.Sprintf("%s", node.Host)

	if node.IP != "" {
		label = fmt.Sprintf("%s (%s)", label, node.IP)
	}

	return label
}


func NodeInArray(array []types.Node, element types.Node) bool {
	contains := false
	for _, v := range array {
		if NodeEquals(v, element) {
			contains = true
			break
		}
	}

	return contains
}