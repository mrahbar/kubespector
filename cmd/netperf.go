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
	"strconv"
	"bytes"
	"text/template"
	"io/ioutil"
	"path/filepath"
)

var (
	outputFile  *os.File
	node        integration.Node
	sshOpts     integration.SSHConfig
	spacesRegex = regexp.MustCompile("[ ]+")
)

type podDeployment struct {
	Name  string
	Image string
	Port  int
}

const (
	testNamespace = "netperf"
	hostName      = "netperf-orchestrator"
	clientName    = "netperf-worker"

	debugLog         = "output.txt"
	csvDataMarker    = "GENERATING CSV OUTPUT"
	csvEndDataMarker = "END CSV DATA"
	netperfImage     = "endianogino/netperf:1.0"

	runUUID          = "latest"
	orchestratorPort = 5202
	iperf3Port       = 5201
	netperfPort      = 12865
)

type port struct {
	Name       string
	Protocol   string
	Port       int
	TargetPort int
}

type env struct {
	Name  string
	Value string
}

type service struct {
	Name      string
	Namespace string
	Ports     []port
}

type ReplicationController struct {
	Name          string
	Namespace     string
	Image         string
	NodeName      string
	ContainerMode string
	ContainerPort int
	ClientPod     bool
}

const (
	NAMESPACE_TEMPLATE = `apiVersion: v1
kind: Namespace
metadata:
  name: {{.Namespace}}
`
	SERVICE_TEMPLATE = `apiVersion: v1
kind: Service
metadata:
  name: {{.Name}}
  labels:
    app: {{.Name}}
  namespace: {{.Namespace}}
spec:
  ports:{{range $i, $a := .Ports}}
  - name: {{.Name}}
    protocol: {{.Protocol}}
    port: {{.Port}}
    targetPort: {{.TargetPort}}{{end}}
  type: ClientIP
`

	RC_TEMPLATE = `apiVersion: v1
kind: ReplicationController
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
spec:
  replicas: 1
  selector:
    app: {{.Name}}
  template:
    metadata:
      name: {{.Name}}
      labels:
        app: {{.Name}}
    spec:{{if .NodeName }}
      nodeName: {{.NodeName}}{{end}}
      containers:
      - name: {{.Name}}
        image: {{.Image}}
        imagePullPolicy: Always
        args:
        - --mode={{.ContainerMode}}{{if .ContainerPort }}
        ports:
        - containerPort: {{.ContainerPort}}{{end}}{{if .ClientPod }}
        env:
        - name: kubeNode
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: worker
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: HOSTNAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name{{end}}
`
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
	Use:   "net",
	Short: "Runs netperf tests on a cluster",
	Long:  `This is a tool for running netperf tests on a cluster. The cluster should have two worker nodes.`,
	Run:   netperfRun,
}

func init() {
	PerfCmd.AddCommand(netperfCmd)
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

	checkingPreconditions()
	createTestNamespace()
	createServices()
	createReplicationControllers()

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

func checkingPreconditions() {
	template := "\"{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}\""
	args := []string{"get", "nodes", "-o", "jsonpath=" + template, "|", "tr", "';'", "\\n", "|", "grep", "\"Ready=True\"", "|", "wc", "-l"}
	result, err := runKubectlCommand(args)

	if err != nil {
		integration.PrettyPrintErr(out, "Error checking node count:\n\tResult: %s\tErr: %s\n", result, err)
		os.Exit(1)
	} else {
		count, errAtoi := strconv.Atoi(strings.TrimRight(result, "\n"))

		if errAtoi != nil {
			integration.PrettyPrintErr(out, "Error converting node count:\n\tErr: %s\n", errAtoi)
			os.Exit(1)
		} else if count < 2 {
			integration.PrettyPrintErr(out, "Insufficient number of nodes for netperf test (need minimum 2 nodes)")
			os.Exit(1)
		}
	}
}

func createTestNamespace() {
	data := make(map[string]string)
	data["Namespace"] = testNamespace
	result, err := deployKubernetesResource(NAMESPACE_TEMPLATE, data)

	if err != nil {
		integration.PrettyPrintErr(out, "Error creating test namespace:\n\tResult: %s\tErr: %s\n", result, err)
		os.Exit(1)
	}
}

func createServices() {
	// Host
	data := service{Name: hostName, Namespace: testNamespace, Ports: []port{
		{
			Name:       hostName,
			Port:       orchestratorPort,
			Protocol:   "TCP",
			TargetPort: orchestratorPort,
		},
	}}
	createService(hostName, data)

	// Client
	data = service{Name: clientName, Namespace: testNamespace, Ports: []port{
		{
			Name:       "netperf-w2",
			Protocol:   "TCP",
			Port:       iperf3Port,
			TargetPort: iperf3Port,
		},
		{
			Name:       "netperf-w2-udp",
			Protocol:   "UDP",
			Port:       iperf3Port,
			TargetPort: iperf3Port,
		},
		{
			Name:       "netperf-w2-netperf",
			Protocol:   "TCP",
			Port:       netperfPort,
			TargetPort: netperfPort,
		},
	}}
	createService(clientName, data)
}

func createService(name string, serviceData interface{}) {
	result, err := deployKubernetesResource(SERVICE_TEMPLATE, serviceData)

	if err != nil {
		if strings.Contains(result, "AlreadyExists") {
			integration.PrettyPrintIgnored(out, "Service: %s already exists.\n", name)
		} else {
			integration.PrettyPrintErr(out, "Error adding service %v:\n\tResult: %s\tErr: %s\n", name, result, err)
			os.Exit(1)
		}
	} else {
		integration.PrettyPrint(out, "Service %s created.", name)
	}
}

func createReplicationControllers() {
	hostRC := ReplicationController{Name: hostName, Namespace: testNamespace, Image: netperfImage, ContainerMode: "orchestrator", ContainerPort: orchestratorPort}
	result, err := deployKubernetesResource(RC_TEMPLATE, hostRC)

	if err != nil {
		integration.PrettyPrintErr(out, "Error creating orchestrator replication controller:\n\tResult: %s\tErr: %s\n", result, err)
		os.Exit(1)
	} else {
		integration.PrettyPrint(out, "Created %s replication-controller", hostName)
	}

	//Created orchestrator replication controller
	clientRC := ReplicationController{Name: clientName, Namespace: testNamespace, Image: netperfImage, ContainerMode: "worker", NodeName: "kubeNode", ClientPod: true}
}

func runKubectlCommand(args []string) (string, error) {
	cmdArgs := strings.Join(args, " ")

	if RootOpts.Debug {
		integration.PrettyPrint(out, "Running kubectl command '%s'\n\n", cmdArgs)
	}

	return integration.PerformSSHCmd(out, sshOpts, node, fmt.Sprintf("kubectl %s", cmdArgs), RootOpts.Debug)
}

func deployKubernetesResource(tpl string, data interface{}) (string, error) {
	var definition bytes.Buffer

	tmpl, _ := template.New("kube-template").Parse(tpl)
	tmpl.Execute(&definition, data)

	tmpFile, err := ioutil.TempFile(os.TempDir(), "kube-")
	if err != nil {
		integration.PrettyPrintErr(out, "Error creating temporary file:\tErr: %s\n", err)
		os.Exit(1)
	}

	defer os.Remove(tmpFile.Name())
	ioutil.WriteFile(tmpFile.Name(), definition.Bytes(), os.ModeAppend)
	remoteFile := path.Join(os.TempDir(), filepath.Base(tmpFile.Name()))
	integration.PerformSCPCmdToRemote(out, sshOpts, node, tmpFile.Name(), remoteFile, RootOpts.Debug)

	args := []string{"apply", "-f", remoteFile}
	result, err := runKubectlCommand(args)
	integration.PerformSSHCmd(out, sshOpts, node, fmt.Sprintf("rm -f %s", remoteFile), RootOpts.Debug)

	return result, err
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
