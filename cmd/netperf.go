package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"regexp"
	"strings"
	"os"
	"time"
	"github.com/spf13/viper"
	"github.com/mrahbar/kubernetes-inspector/integration"

	"path"
)

var (
	outputFile  *os.File
	node        integration.Node
	sshOpts     integration.SSHConfig
	spacesRegex  = regexp.MustCompile("[ ]+")
)

const (
	HOST_NAME   = "netperf-tester-host"
	CLIENT_NAME = "netperf-tester-client"
)

type netperfCliOpts struct {
	output  string
	num     int
	cleanup bool
	verbose bool
}

var netperfOpts = &netperfCliOpts{}

// netperfCmd represents the netperf command
var netperfCmd = &cobra.Command{
	Use:   "netperf",
	Short: "Runs netperf tests on a cluster",
	Long:  `This is a tool for running netperf tests on a cluster. The cluster should have two worker nodes.`,
	Run:   netperfRun,
}

func init() {
	RootCmd.AddCommand(netperfCmd)
	netperfCmd.Flags().StringVarP(&netperfOpts.output, "output", "o", "./netperf.out", "Full path to the csv file to output")
	netperfCmd.Flags().IntVarP(&netperfOpts.num, "num", "n", 1000, "Number of times to run netperf")
	netperfCmd.Flags().BoolVarP(&netperfOpts.cleanup, "cleanup", "c", true, "Delete test pods when done")
	netperfCmd.Flags().BoolVarP(&netperfOpts.verbose, "verbose", "v", true, "Print results to standard out. Use -v=false to turn it off.")
}

func netperfRun(_ *cobra.Command, _ []string) {
	var config integration.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		integration.PrettyPrintErr(out, "Unable to decode config: %v", err)
		os.Exit(1)
	}

	group := integration.FindGroupByName(config.ClusterGroups, integration.MASTER_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		integration.PrettyPrintErr(out, "No host configured for group [%s]", integration.MASTER_GROUPNAME)
		os.Exit(1)
	}

	sshOpts = config.Ssh
	sshOpts.Sudo = false
	node = integration.GetFirstAccessibleNode(group.Nodes, RootOpts.Debug)

	if !integration.IsNodeAddressValid(node) {
		integration.PrettyPrintErr(out, "No master available")
		os.Exit(1)
	}

	if netperfOpts.output == "" {
		ex, err := os.Executable()
		if err != nil {
			os.Exit(1)
		}
		exPath := path.Dir(ex)
		netperfOpts.output = path.Join(exPath, "netperf.out")
	}

	err = error(nil)
	outputFile, err = os.Create(netperfOpts.output)
	if err != nil {
		integration.PrettyPrintErr(out, "Failed to open output file for path %s Error: %v", netperfOpts.output, err)
		os.Exit(1)
	}

	integration.PrettyPrint(out, "Running kubectl commands on node %s\n\n", integration.ToNodeLabel(node))

	// setup pod with server to access from test pod
	addService(HOST_NAME, "paultiplady/netserver:ubuntu.2", 12865)
	// setup test pod
	addService(CLIENT_NAME, "paultiplady/netserver:ubuntu.2", 12865)

	integration.PrettyPrint(out, "Waiting for services to be Running...\n")
	waitForServicesToBeRunning()
	displayTestPods()

	// run the tests
	if err := runTests(); err != nil {
		integration.PrettyPrintWarn(out, "Error running tests: %v\n", err)
	}

	// cleanup services
	if netperfOpts.cleanup {
		integration.PrettyPrint(out, "Cleaning up netperf-test\n")
		removeService(HOST_NAME)
		removeService(CLIENT_NAME)
	}

	integration.PrettyPrintOk(out, "DONE\n")
}

func runKubectlCommand(args []string) (string, error) {
	cmdArgs := strings.Join(args, " ")

	if RootOpts.Debug {
		integration.PrettyPrint(out, "Running kubectl command '%s'\n\n", cmdArgs)
	}

	o, err := integration.PerformSSHCmd(out, sshOpts, node, fmt.Sprintf("kubectl %s", cmdArgs), RootOpts.Debug)
	result := strings.TrimSpace(o)

	return result, err
}

func addService(name, image string, port int) {
	integration.PrettyPrint(out, "Adding pod: %s\n", name)
	args := []string{"run", name, "--image=" + image, fmt.Sprintf("--port=%d", port), "--hostport=65530"}
	result, err := runKubectlCommand(args)

	if err != nil {
		if strings.Contains(result, "AlreadyExists") {
			integration.PrettyPrintIgnored(out, "Service: %s already exists.\n", name)
		} else {
			integration.PrettyPrintErr(out, "Error adding service %v:\n\tResult: %s\tErr: %s\n", name, result, err)
			os.Exit(1)
		}
	}
}

func waitForServicesToBeRunning() {
	waitTime := time.Second
	done := false
	for !done {
		template := "\"{..status.phase}\""
		args := []string{"get", "pods", "-o", "jsonpath=" + template}
		result, err := runKubectlCommand(args)

		if err != nil {
			integration.PrettyPrintWarn(out, "Error running kubectl command '%v':\n\tResult: %s\tErr: %s\n", args, result, err)
		}

		lines := strings.Split(result, " ")
		if len(lines) < 2 {
			integration.PrettyPrint(out, "Service status output too short. Waiting %v then checking again.\n", waitTime)
			time.Sleep(waitTime)
			waitTime *= 2
			continue
		}
		if lines[0] != "Running" || lines[1] != "Running" {
			integration.PrettyPrint(out, "Services not running. Waiting %v then checking again.\n", waitTime)
			time.Sleep(waitTime)
			waitTime *= 2
		} else {
			done = true
		}
	}
}

func displayTestPods() {
	result, err := runKubectlCommand([]string{"get", "pods", "-o=wide"})
	integration.PrettyPrint(out, "Pods are running:\n%s\n", result)

	if err != nil {
		integration.PrettyPrintWarn(out, "Error running kubectl command '%v'\n", err)
	}
	integration.PrettyPrint(out, "\n")
}

func removeService(name string) {
	_, err := runKubectlCommand([]string{"delete", "deployments/" + name})

	if err != nil {
		integration.PrettyPrintWarn(out, "Error running kubectl command '%v'\n", err)
	}
}

func runTests() error {
	// get client pod name
	clientName, err := getPodName(CLIENT_NAME)
	if clientName == "" || err != nil {
		return err
	}

	// get ip of the host pod
	hostIP, err := getPodIP(HOST_NAME)
	if hostIP == "" || err != nil {
		return err
	}

	integration.PrettyPrint(out, "Running netperf tests %d times.\n\n", netperfOpts.num)

	for i := 0; i < netperfOpts.num; i++ {
		runTest(clientName, hostIP, i)
	}

	integration.PrettyPrint(out, "\n")
	return nil
}

func getPodName(name string) (string, error) {
	template := "\"{..metadata.name}\""
	args := []string{"get", "pods", "-l", "run=" + name, "-o", "jsonpath=" + template}
	result, err := runKubectlCommand(args)

	if err != nil {
		return "", err
	}
	return strings.TrimRight(result, "\n"), nil
}

func getPodIP(name string) (string, error) {
	template := "\"{..status.podIP}\""
	args := []string{"get", "pods", "-l", "run=" + name, "-o", "jsonpath=" + template}
	result, err := runKubectlCommand(args)

	if err != nil {
		return "", err
	}

	return strings.Trim(result, " \n"), nil
}

func runTest(clientName, hostIP string, testNumber int) error {
	args := []string{"exec", "-t", clientName, "--", "netperf", "-H", hostIP, "-j", "-c", "-l", "-1000", "-t", "TCP_RR"}
	if testNumber != 0 {
		args = append(args, "-P", "0")
	}

	args = append(args, "--", "-D", "-O", "THROUGHPUT_UNITS,THROUGHPUT,MEAN_LATENCY,MIN_LATENCY,MAX_LATENCY,P50_LATENCY,P90_LATENCY,P99_LATENCY,STDDEV_LATENCY,LOCAL_CPU_UTIL")
	result, err := runKubectlCommand(args)
	if err != nil {
		integration.PrettyPrintWarn(out, "Error running command '%v':\n\tResult: %s\tErr: %s\n", args, result, err)
		return err
	}

	if netperfOpts.verbose || RootOpts.Debug {
		integration.PrettyPrint(out, "%s\n", result)
	}

	if outputFile != nil {
		outputFile.WriteString(resultsToCSV(result, testNumber))
	}

	return nil
}

func resultsToCSV(results string, testNumber int) string {
	ret := ""
	line := ""

	if testNumber == 0 {
		ret = "Test #,Throughput Units,Throughput,Mean Latency Microseconds,Minimum Latency Microseconds,Maximum Latency Microseconds,50th Percentile Latency Microseconds,90th Percentile Latency Microseconds,99th Percentile Latency Microseconds,Stddev Latency Microseconds,Local CPU Util %\n"
		lines := strings.SplitN(results, "\n", -1)
		line = lines[len(lines)-2] + "\n"
	} else {
		line = results
	}

	csvLine := spacesRegex.ReplaceAllLiteralString(line, ",")
	csvLine = strings.Replace(csvLine, "\r", "", -1)
	ret += fmt.Sprintf("%d, ", testNumber+1)
	ret += strings.TrimSuffix(csvLine, ",\n") + "\n"

	return ret
}
