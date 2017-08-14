package integration

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"strings"
)

func UnmarshalConfig() types.Config {
	var config types.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		PrettyPrintErr("Unable to decode config: %v", err)
		os.Exit(1)
	}

	return config
}

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

func FindGroupByName(clustergroups []types.ClusterGroup, name string) types.ClusterGroup {
	for _, group := range clustergroups {
		if strings.EqualFold(group.Name, name) {
			return group
		}
	}

	return types.ClusterGroup{}
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

func GetFirstAccessibleNode(localOn types.Node, nodes []types.Node, debug bool) types.Node {
	if IsNodeAddressValid(localOn) {
		for _, n := range nodes {
			if NodeEquals(localOn, n) {
				return n
			}
		}
	}

	for _, n := range nodes {
		nodeAddress := GetNodeAddress(n)

		result, err := Ping(nodeAddress, n.Host)
		if debug {
			fmt.Printf("Result for ping on %s:\n\tResult: %s\tErr: %s\n", n.Host, result, err)
		}
		if err == nil {
			return n
		}
	}

	return nil
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
