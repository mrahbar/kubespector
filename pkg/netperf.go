package pkg

import (
	"bytes"
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/types"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var (
	outputFile *os.File
	node       types.Node
	sshOpts    types.SSHConfig
)

type podDeployment struct {
	Name  string
	Image string
	Port  int
}

const (
	testNamespace    = "netperf"
	orchestratorName = "netperf-orchestrator"
	workerName       = "netperf-worker"

	orchestratorMode = "orchestrator"
	workerMode       = "worker"

	csvDataMarker    = "GENERATING CSV OUTPUT"
	csvEndDataMarker = "END CSV DATA"
	netperfImage     = "endianogino/netperf:1.1"

	workerCount      = 3
	orchestratorPort = 5202
	iperf3Port       = 5201
	netperfPort      = 12865
)

type servicePort struct {
	Name       string
	Protocol   string
	Port       int
	TargetPort int
}

type podPort struct {
	Name     string
	Protocol string
	Port     int
}

type env struct {
	Name  string
	Value string
}

type service struct {
	Name      string
	Namespace string
	Ports     []servicePort
}

type ReplicationController struct {
	Name              string
	Namespace         string
	Image             string
	NodeName          string
	ContainerMode     string
	Ports             []podPort
	HostIP            string
	ClientPod         bool
	OrchestratorPodIP string
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
  selector:
    app: {{.Name}}
  type: ClusterIP
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
        - --mode={{.ContainerMode}}
        {{- if .Ports }}
		ports:{{range $i, $a := .Ports}}
        - name: {{.Name}}
          protocol: {{.Protocol}}
          port: {{.Port}}{{end}}{{end}}
		{{- if .ClientPod }}
        env:
        - name: workerPodIP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        - name: workerName
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: orchestratorPort
          value: 5202
        - name: orchestratorPodIP
          value: {{.OrchestratorPodIP}}{{end}}
`
)

var netperfOpts *types.NetperfOpts

func Netperf(config types.Config, opts *types.NetperfOpts) {
	netperfOpts = opts
	group := integration.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		integration.PrettyPrintErr("No host configured for group [%s]", types.MASTER_GROUPNAME)
		os.Exit(1)
	}

	sshOpts = config.Ssh
	sshOpts.Sudo = false
	node = integration.GetFirstAccessibleNode(group.Nodes, netperfOpts.Debug)

	if !integration.IsNodeAddressValid(node) {
		integration.PrettyPrintErr("No master available")
		os.Exit(1)
	}

	if netperfOpts.Output == "" {
		ex, err := os.Executable()
		if err != nil {
			os.Exit(1)
		}
		exPath := path.Dir(ex)
		netperfOpts.Output = path.Join(exPath, "netperf.out")
	}

	err := error(nil)
	outputFile, err = os.Create(netperfOpts.Output)
	if err != nil {
		integration.PrettyPrintErr("Failed to open output file for path %s Error: %v", netperfOpts.Output, err)
		os.Exit(1)
	}

	integration.PrettyPrint("Running kubectl commands on node %s\n\n", integration.ToNodeLabel(node))

	checkingPreconditions()
	createTestNamespace()
	createServices()
	createReplicationControllers()

	integration.PrettyPrint("Waiting for pods to be Running...\n")
	waitForServicesToBeRunning()
	displayTestPods()

	fetchTestResults()

	// cleanup services
	if netperfOpts.Cleanup {
		integration.PrettyPrint("Cleaning up...\n")
		removeServices()
		removeReplicationControllers()
	}

	integration.PrettyPrintOk("DONE\n")
}

func checkingPreconditions() {
	tmpl := "\"{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}\""
	args := []string{"get", "nodes", "-o", "jsonpath=" + tmpl, " | ", "tr", "';'", "\\\\n", " | ", "grep", "\"Ready=True\"", " | ", "wc", "-l"}
	result, err := runKubectlCommand(args)

	if err != nil {
		integration.PrettyPrintErr("Error checking node count:\n\tResult: %s\tErr: %s\n", result, err)
		os.Exit(1)
	} else {
		count, errAtoi := strconv.Atoi(strings.TrimRight(result, "\n"))

		if errAtoi != nil {
			integration.PrettyPrintErr("Error getting node count:\n\tErr: %s\n", errAtoi)
			os.Exit(1)
		} else if count < 2 {
			integration.PrettyPrintErr("Insufficient number of nodes for netperf test (need minimum 2 nodes)")
			os.Exit(1)
		}
	}
}

func createTestNamespace() {
	integration.PrettyPrint("Creating namespace\n")
	data := make(map[string]string)
	data["Namespace"] = testNamespace
	result, err := deployKubernetesResource(NAMESPACE_TEMPLATE, data)

	if err != nil {
		integration.PrettyPrintErr("Error creating test namespace:\n\tResult: %s\tErr: %s\n", result, err)
		os.Exit(1)
	} else {
		integration.PrettyPrint("Namespace %s created\n", testNamespace)
	}
	integration.PrettyPrint("\n")
}

func createServices() {
	integration.PrettyPrint("Creating services\n")
	// Host
	data := service{Name: orchestratorName, Namespace: testNamespace, Ports: []servicePort{
		{
			Name:       orchestratorName,
			Port:       orchestratorPort,
			Protocol:   "TCP",
			TargetPort: orchestratorPort,
		},
	}}
	createService(orchestratorName, data)

	// Create the netperf-w2 service that points a clusterIP at the worker 2 pod
	name := fmt.Sprintf("%s-%d", workerName, 2)
	data = service{Name: name, Namespace: testNamespace, Ports: []servicePort{
		{
			Name:       name,
			Protocol:   "TCP",
			Port:       iperf3Port,
			TargetPort: iperf3Port,
		},
		{
			Name:       fmt.Sprintf("%s-%s", name, "udp"),
			Protocol:   "UDP",
			Port:       iperf3Port,
			TargetPort: iperf3Port,
		},
		{
			Name:       fmt.Sprintf("%s-%s", name, "netperf"),
			Protocol:   "TCP",
			Port:       netperfPort,
			TargetPort: netperfPort,
		},
	}}
	createService(name, data)
	integration.PrettyPrint("\n")
}

func createService(name string, serviceData interface{}) {
	result, err := deployKubernetesResource(SERVICE_TEMPLATE, serviceData)

	if err != nil {
		if strings.Contains(result, "AlreadyExists") {
			integration.PrettyPrintIgnored("Service: %s already exists.\n", name)
		} else {
			integration.PrettyPrintErr("Error adding service %v:\n\tResult: %s\tErr: %s\n", name, result, err)
			os.Exit(1)
		}
	} else {
		integration.PrettyPrint("Service %s created.\n", name)
	}
}

func createReplicationControllers() {
	integration.PrettyPrint("Creating ReplicationControllers\n")

	hostRC := ReplicationController{Name: orchestratorName, Namespace: testNamespace,
		Image:                            netperfImage, ContainerMode: orchestratorMode, Ports: []podPort{
			{
				Name:     "rpc-service-port",
				Protocol: "TCP",
				Port:     orchestratorPort,
			},
		},
	}
	result, err := deployKubernetesResource(RC_TEMPLATE, hostRC)

	if err != nil {
		integration.PrettyPrintErr("Error creating %s replication controller:\n\tResult: %s\tErr: %s\n", orchestratorName, result, err)
		os.Exit(1)
	} else {
		integration.PrettyPrint("Created %s replication-controller\n", orchestratorName)
	}

	args := []string{"get", "nodes", " | ", "grep", "-w", "\"Ready\"", " | ", "sed", "-e", "\"s/[[:space:]]\\+/,/g\""}
	result, err = runKubectlCommand(args)

	if err != nil {
		integration.PrettyPrintErr("Error getting nodes for worker replication controller:\n\tResult: %s\tErr: %s\n", result, err)
		os.Exit(1)
	} else {
		hostIP, err := getServiceIP(orchestratorName)
		if hostIP == "" || err != nil {
			integration.PrettyPrintErr("Error getting clusterIP of service %s:\n\tResult: %s\tErr: %s\n", orchestratorName, result, err)
			os.Exit(1)
		}

		lines := strings.SplitN(result, "\n", -1)
		firstNode := strings.Split(lines[0], ",")[0]
		secondNode := strings.Split(lines[1], ",")[0]

		// wait a little to give orchestrator pod time to start
		time.Sleep(5 * time.Second)
		orchestratorPodIP := getPodIP(orchestratorName) //TODO test with service dns instead

		for i := 1; i <= workerCount; i++ {
			name := fmt.Sprintf("%s-%d", workerName, i)
			kubeNode := firstNode
			if i == 3 {
				kubeNode = secondNode
			}

			clientRC := ReplicationController{Name: name, Namespace: testNamespace, Image: netperfImage,
				ContainerMode:                      workerMode, HostIP: hostIP, NodeName: kubeNode, ClientPod: true, OrchestratorPodIP: orchestratorPodIP,
				Ports: []podPort{
					{
						Name:     "iperf3-server-port",
						Protocol: "UDP",
						Port:     iperf3Port,
					},
					{
						Name:     "netperf-server-port",
						Protocol: "TCP",
						Port:     netperfPort,
					},
				},
			}

			result, err := deployKubernetesResource(RC_TEMPLATE, clientRC)

			if err != nil {
				integration.PrettyPrintErr("Error creating %s replication controller:\n\tResult: %s\tErr: %s\n", name, result, err)
				os.Exit(1)
			} else {
				integration.PrettyPrint("Created %s replication-controller\n", name)
			}
		}
	}
	integration.PrettyPrint("\n")
}

func runKubectlCommand(args []string) (string, error) {
	a := strings.Join(args, " ")
	return integration.PerformSSHCmd(sshOpts, node, fmt.Sprintf("kubectl %s", a), netperfOpts.Debug)
}

func deployKubernetesResource(tpl string, data interface{}) (string, error) {
	var definition bytes.Buffer

	tmpl, _ := template.New("kube-template").Parse(tpl)
	tmpl.Execute(&definition, data)

	tmpFile, err := ioutil.TempFile(os.TempDir(), "kube-")
	if err != nil {
		integration.PrettyPrintErr("Error creating temporary file:\tErr: %s\n", err)
		os.Exit(1)
	}

	defer os.Remove(tmpFile.Name())
	ioutil.WriteFile(tmpFile.Name(), definition.Bytes(), os.ModeAppend)
	remoteFile := path.Join(os.TempDir(), filepath.Base(tmpFile.Name()))
	integration.PerformSCPCmdToRemote(sshOpts, node, tmpFile.Name(), remoteFile, netperfOpts.Debug)

	args := []string{"apply", "-f", remoteFile}
	result, err := runKubectlCommand(args)
	integration.PerformSSHCmd(sshOpts, node, fmt.Sprintf("rm -f %s", remoteFile), netperfOpts.Debug)

	return result, err
}

func waitForServicesToBeRunning() {
	waitTime := time.Second
	done := false
	for !done {
		tmpl := "\"{..status.phase}\""
		args := []string{"--namespace=" + testNamespace, "get", "pods", "-o", "jsonpath=" + tmpl}
		result, err := runKubectlCommand(args)

		if err != nil {
			integration.PrettyPrintWarn("Error running kubectl command '%v':\n\tResult: %s\tErr: %s\n", args, result, err)
		}

		lines := strings.Split(result, " ")
		if len(lines) < workerCount+1 {
			integration.PrettyPrint("Service status output too short. Waiting %v then checking again.\n", waitTime)
			time.Sleep(waitTime)
			waitTime *= 2
			continue
		}
		if lines[0] != "Running" || lines[1] != "Running" {
			integration.PrettyPrint("Services not running. Waiting %v then checking again.\n", waitTime)
			time.Sleep(waitTime)
			waitTime *= 2
		} else {
			done = true
		}
	}
	integration.PrettyPrint("\n")
}

func displayTestPods() {
	result, err := runKubectlCommand([]string{"--namespace=" + testNamespace, "get", "pods", "-o=wide"})
	integration.PrettyPrint("Pods are running:\n%s\n", result)

	if err != nil {
		integration.PrettyPrintWarn("Error running kubectl command '%v'\n", err)
	}

	integration.PrettyPrint("\n")
}

func fetchTestResults() {
	orchestratorPodName := getPodName(orchestratorName)
	sleep := 30 * time.Second

	for len(orchestratorPodName) == 0 {
		integration.PrettyPrint("Waiting %s for orchestrator pod creation\n", sleep)
		time.Sleep(sleep)
		orchestratorPodName = getPodName(orchestratorName)
	}
	fmt.Println("Orchestrator Pod is", orchestratorPodName)

	// The pods orchestrate themselves, we just wait for the results file to show up in the orchestrator container
	for true {
		// Monitor the orchestrator pod for the CSV results file
		csvdata := getCsvResultsFromPod(orchestratorPodName)
		if csvdata == nil {
			integration.PrettyPrint("Scanned orchestrator pod filesystem - no results file found yet...waiting %s for orchestrator to write CSV file...\n", sleep)
			time.Sleep(sleep)
			continue
		}
		integration.PrettyPrint("Test concluded - CSV raw data written to %s.csv\n", netperfOpts.Output)
		if processCsvData(csvdata) {
			break
		}
	}
}

// Retrieve the logs for the pod/container and check if csv data has been generated
func getCsvResultsFromPod(podName string) *string {
	logData, err := runKubectlCommand([]string{"--namespace=" + testNamespace, "logs", podName, "--timestamps=false"})
	if err != nil {
		integration.PrettyPrintWarn("Error reading logs from pod %s:\n\tResult: %s\tErr: %s\n", podName, logData, err)
		return nil
	}

	// TODO maybe download OutputCaptureFile = "/tmp/output.txt" instead
	index := strings.Index(logData, csvDataMarker)
	endIndex := strings.Index(logData, csvEndDataMarker)
	if index == -1 || endIndex == -1 {
		return nil
	}

	csvData := string(logData[index+len(csvDataMarker)+1: endIndex])
	return &csvData
}

// processCsvData : Process the CSV datafile and generate line and bar graphs
func processCsvData(csvData *string) bool {
	_, err := outputFile.WriteString(*csvData)
	outputFile.Close()
	if err != nil {
		integration.PrettyPrintErr("ERROR writing output CSV datafile: %s", err)
		return false
	}

	return true
}

func removeServices() {
	removeService(orchestratorName)
	removeService(fmt.Sprintf("%s-%d", workerName, 2))
}

func removeService(name string) {
	_, err := runKubectlCommand([]string{"--namespace=" + testNamespace, "delete", "svc/" + name})

	if err != nil {
		integration.PrettyPrintWarn("Error deleting service '%v'\n", name, err)
	}
}

func removeReplicationControllers() {
	removeReplicationController(orchestratorName)
	for i := 1; i <= workerCount; i++ {
		removeReplicationController(fmt.Sprintf("%s-%d", workerName, i))
	}
}

func removeReplicationController(name string) {
	_, err := runKubectlCommand([]string{"--namespace=" + testNamespace, "delete", "rc/" + name})

	if err != nil {
		integration.PrettyPrintWarn("Error deleting replication-controller '%v'\n", name, err)
	}
}

func getPodName(name string) string {
	tmpl := "\"{..metadata.name}\""
	args := []string{"--namespace=" + testNamespace, "get", "pods", "-l", "app=" + name, "-o", "jsonpath=" + tmpl}
	result, err := runKubectlCommand(args)

	if err != nil {
		return ""
	}

	return strings.TrimRight(result, "\n")
}

func getPodIP(name string) string {
	tmpl := "\"{..status.podIP}\""
	args := []string{"--namespace=" + testNamespace, "get", "pods", "-l", "app=" + name, "-o", "jsonpath=" + tmpl}
	result, err := runKubectlCommand(args)

	if err != nil {
		return ""
	}

	return strings.TrimRight(result, "\n")
}

func getServiceIP(name string) (string, error) {
	template := "\"{..spec.clusterIP}\""
	args := []string{"--namespace=" + testNamespace, "get", "service", "-l", "app=" + name, "-o", "jsonpath=" + template}
	result, err := runKubectlCommand(args)

	if err != nil {
		return "", err
	}

	return strings.Trim(result, " \n"), nil
}
