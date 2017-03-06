package cmd

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/spf13/cobra"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/util"
	"strings"
	"os"
	"io"
)

var groups = []string{"Etcd", "Master", "Worker", "Kubernetes"}
var groupsArg string
var out io.Writer = os.Stdout

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of Kubernetes services defined in configuration file",
	Long: `When called without arguments all hosts in configuration will be examined.`,
	Run: statusRun,
}

func init() {
	RootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVarP(&groupsArg, "groups", "g", "",  "Comma-separated list of group names")
}

func statusRun(cmd *cobra.Command, args []string) {
	var config integration.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		fmt.Printf("Unable to decode into struct, %v\n", err)
	} else {
		if  groupsArg != "" {
			groups = strings.Split(groupsArg, ",")
			fmt.Printf("Restricted status check to groups: %v\n", strings.Join(groups, " "))
		} else {
			fmt.Printf("Performing status check for groups: %v\n", strings.Join(groups, " "))
		}

		for _, element := range groups {
			switch element {
			case "Etcd":
				checkStatus(&config.Ssh, element, config.Cluster.Etcd.Services, config.Cluster.Etcd.Nodes)
			case "Master":
				checkStatus(&config.Ssh, element, config.Cluster.Master.Services, config.Cluster.Master.Nodes)
			case "Worker":
				checkStatus(&config.Ssh, element, config.Cluster.Worker.Services, config.Cluster.Worker.Nodes)
			case "Kubernetes":
				checkKubernetesStatus(&config.Ssh, element, config.Kubernetes.Resources, config.Cluster.Master.Nodes)
			}
		}
	}
}

func checkStatus(sshOpts *integration.SSHConfig, element string, services []string, nodes []integration.Node) {
	util.PrintHeader(out, fmt.Sprintf("Checking status of [%s] ", element), '=')
	if nodes == nil || len(nodes) == 0  {
		util.PrettyPrintWarn(out, "No host configured for [%s]", element)
		return
	}
	if services == nil || len(services) == 0   {
		util.PrettyPrintWarn(out, "No services configured for [%s]", element)
		return
	}

	for _, node := range nodes {
		for _, service := range services {
			util.PrettyPrint(out, "Status of %s on host %s (%s):\n", service, node.Host, node.IP)
			o, err := doSSH(sshOpts, &node, fmt.Sprintf("sudo systemctl status %s | grep Active:", service))

			if err != nil {
				util.PrettyPrintErr(out, "Error checking status of %s: %v", service, node.Host, node.IP, err)
			} else {
				util.PrettyPrintOk(out, strings.Trim(o, " "))
			}
		}
	}
}

func checkKubernetesStatus(sshOpts *integration.SSHConfig, element string,
	resources []integration.KubernetesResource, nodes []integration.Node)  {
	util.PrintHeader(out, fmt.Sprintf("Checking status of [%s] ", element), '=')

	if nodes == nil || len(nodes) == 0  {
		util.PrettyPrintWarn(out, "No master host configured for [%s]", element)
		return
	}
	if resources == nil || len(resources) == 0   {
		util.PrettyPrintWarn(out, "No resources configured for [%s]", element)
		return
	}

	node := nodes[0]
	for _, resource := range resources {
		msg := fmt.Sprintf("Status of %s", resource.Type)
		command := fmt.Sprintf("sudo kubectl get %s", resource.Type)
		if resource.Namespace != "" {
			msg += " in namespace: " + resource.Namespace
			command += " -n " + resource.Namespace
		}
		if resource.Wide {
			command += " -o wide"
		}

		util.PrettyPrint(out, msg+"\n")
		o, err := doSSH(sshOpts, &node, command)

		if err != nil {
			util.PrettyPrintErr(out, "Error checking %s in namespace %s on host %s (%s): %v",
				resource.Type, resource.Namespace, node.Host, node.IP, err)
		} else {
			util.PrettyPrintOk(out, o)
		}
	}
}

func doSSH(sshOpts *integration.SSHConfig, node *integration.Node, cmd string) (string ,error) {
	client, err := integration.NewClient(node.IP, sshOpts.Port, sshOpts.User, sshOpts.Key,
		strings.FieldsFunc(sshOpts.Options, func(r rune) bool {
			return r == ' ' || r == ','
		}))

	if err != nil {
		msg := fmt.Sprintf("Error creating SSH client for host %s (%s): %v", node.Host, node.IP, err)
		util.PrettyPrintErr(out, msg)
		return "", err
	}

	return client.Output(sshOpts.Pty, cmd)
}