package integration

import (
	"fmt"
	"strings"
	"github.com/spf13/pflag"
	"github.com/spf13/cobra"
)

func GetNodeAddress(node Node) string {
	nodeAddress := node.IP
	if nodeAddress == "" {
		nodeAddress = node.Host
	}

	return nodeAddress
}

func IsNodeAddressValid(node Node) bool {
	if node.Host == "" && node.IP == "" {
		return false
	} else {
		return true
	}
}

func ToNodeLabel(node Node) string {
	if !IsNodeAddressValid(node) {
		return ""
	}

	label := fmt.Sprintf("%s", node.Host)

	if node.IP != "" {
		label = fmt.Sprintf("%s (%s)", label, node.IP)
	}

	return label
}

func FindGroupByName(clustergroups []ClusterGroup, name string) ClusterGroup {
	for _, group := range clustergroups {
		if strings.EqualFold(group.Name, name) {
			return group
		}
	}

	return ClusterGroup{}
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

func GetFirstAccessibleNode(nodes []Node, debug bool) Node {
	var node Node

	for _, n := range nodes {
		nodeAddress := GetNodeAddress(n)

		result, err := Ping(nodeAddress, n.Host)
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

func CheckRequiredFlags(cmd *cobra.Command, _ []string) error {
	f := cmd.Flags()
	requiredError := false
	flagName := ""

	f.VisitAll(func(flag *pflag.Flag) {
		requiredAnnotation := flag.Annotations[cobra.BashCompOneRequiredFlag]
		if len(requiredAnnotation) == 0 {
			return
		}

		flagRequired := requiredAnnotation[0] == "true"

		if flagRequired && !flag.Changed {
			requiredError = true
			flagName = flag.Name
		}
	})

	if requiredError {
		return fmt.Errorf("Required flag `%s` has not been set", flagName)
	}

	return nil
}
