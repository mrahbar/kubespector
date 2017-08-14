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
	workerName       = "netperf-w"

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
          containerPort: {{.Port}}{{end}}{{end}}
		{{- if .ClientPod }}
        env:
        - name: workerPodIP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        - name: workerName
          value: {{.Name}}
        - name: orchestratorPort
          value: "5202"
        - name: orchestratorPodIP
          value: "{{.OrchestratorPodIP}}"{{end}}
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
	node = integration.GetFirstAccessibleNode(sshOpts.LocalOn, group.Nodes, netperfOpts.Debug)

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
	_, err = os.Create(netperfOpts.Output)
	if err != nil {
		integration.PrettyPrintErr("Failed to open output file for path %s Error: %v", netperfOpts.Output, err)
		os.Exit(1)
	}

	integration.PrettyPrint("Running kubectl commands on node %s\n", integration.ToNodeLabel(node))

	checkingPreconditions()
	createTestNamespace()
	createServices()
	createReplicationControllers()

	integration.PrettyPrint("Waiting for pods to be Running...")
	waitForServicesToBeRunning()
	displayTestPods()

	integration.PrettyPrint("Waiting till pods orchestrate themselves. This may take several minutes..")
	fetchTestResults()

	// cleanup services
	if netperfOpts.Cleanup {
		integration.PrettyPrint("Cleaning up...")
		removeServices()
		removeReplicationControllers()
	}

	integration.PrettyPrintOk("DONE")
}

func checkingPreconditions() {
	tmpl := "\"{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}\""
	args := []string{"get", "nodes", "-o", "jsonpath=" + tmpl, " | ", "tr", "';'", "\\\\n", " | ", "grep", "\"Ready=True\"", " | ", "wc", "-l"}
	result, err := runKubectlCommand(args)

	if err != nil {
		integration.PrettyPrintErr("Error checking node count:\n\tResult: %s\tErr: %s", result, err)
		os.Exit(1)
	} else {
		count, errAtoi := strconv.Atoi(strings.TrimRight(result, "\n"))

		if errAtoi != nil {
			integration.PrettyPrintErr("Error getting node count:\n\tErr: %s", errAtoi)
			os.Exit(1)
		} else if count < 2 {
			integration.PrettyPrintErr("Insufficient number of nodes for netperf test (need minimum 2 nodes)")
			os.Exit(1)
		}
	}
}

func createTestNamespace() {
	integration.PrettyPrint("Creating namespace")
	data := make(map[string]string)
	data["Namespace"] = testNamespace
	result, err := deployKubernetesResource(NAMESPACE_TEMPLATE, data)

	if err != nil {
		integration.PrettyPrintErr("Error creating test namespace:\n\tResult: %s\tErr: %s", result, err)
		os.Exit(1)
	} else {
		integration.PrettyPrint("Namespace %s created", testNamespace)
	}
	integration.PrettyPrint("")
}

func createServices() {
	integration.PrettyPrint("Creating services")
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
	name := fmt.Sprintf("%s%d", workerName, 2)
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
	integration.PrettyPrint("")
}

func createService(name string, serviceData interface{}) {
	result, err := deployKubernetesResource(SERVICE_TEMPLATE, serviceData)

	if err != nil {
		if strings.Contains(result, "AlreadyExists") {
			integration.PrettyPrintIgnored("Service: %s already exists.", name)
		} else {
			integration.PrettyPrintErr("Error adding service %v:\n\tResult: %s\tErr: %s", name, result, err)
			os.Exit(1)
		}
	} else {
		integration.PrettyPrint("Service %s created.", name)
	}
}

func createReplicationControllers() {
	integration.PrettyPrint("Creating ReplicationControllers")

	hostRC := ReplicationController{Name: orchestratorName, Namespace: testNamespace,
		Image:                            netperfImage, ContainerMode: orchestratorMode, Ports: []podPort{
			{
				Name:     "service-port",
				Protocol: "TCP",
				Port:     orchestratorPort,
			},
		},
	}
	result, err := deployKubernetesResource(RC_TEMPLATE, hostRC)

	if err != nil {
		integration.PrettyPrintErr("Error creating %s replication controller:\n\tResult: %s\tErr: %s", orchestratorName, result, err)
		os.Exit(1)
	} else {
		integration.PrettyPrint("Created %s replication-controller", orchestratorName)
	}

	args := []string{"get", "nodes", " | ", "grep", "-w", "\"Ready\"", " | ", "sed", "-e", "\"s/[[:space:]]\\+/,/g\""}
	result, err = runKubectlCommand(args)

	if err != nil {
		integration.PrettyPrintErr("Error getting nodes for worker replication controller:\n\tResult: %s\tErr: %s", result, err)
		os.Exit(1)
	} else {
		integration.PrettyPrint("Waiting 5s to give orchestrator pod time to start")
		time.Sleep(5 * time.Second)
		hostIP, err := getServiceIP(orchestratorName)
		if hostIP == "" || err != nil {
			integration.PrettyPrintErr("Error getting clusterIP of service %s:\n\tResult: %s\tErr: %s", orchestratorName, result, err)
			os.Exit(1)
		}

		lines := strings.SplitN(result, "\n", -1)
		firstNode := strings.Split(lines[0], ",")[0]
		secondNode := strings.Split(lines[1], ",")[0]

		for i := 1; i <= workerCount; i++ {
			name := fmt.Sprintf("%s%d", workerName, i)
			kubeNode := firstNode
			if i == 3 {
				kubeNode = secondNode
			}

			clientRC := ReplicationController{Name: name, Namespace: testNamespace, Image: netperfImage,
				ContainerMode:                      workerMode, NodeName: kubeNode, ClientPod: true, OrchestratorPodIP: hostIP,
				Ports: []podPort{
					{
						Name:     "iperf3-port",
						Protocol: "UDP",
						Port:     iperf3Port,
					},
					{
						Name:     "netperf-port",
						Protocol: "TCP",
						Port:     netperfPort,
					},
				},
			}

			result, err := deployKubernetesResource(RC_TEMPLATE, clientRC)

			if err != nil {
				integration.PrettyPrintErr("Error creating %s replication controller:\n\tResult: %s\tErr: %s", name, result, err)
				os.Exit(1)
			} else {
				integration.PrettyPrint("Created %s replication-controller", name)
			}
		}
	}
	integration.PrettyPrint("")
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
		integration.PrettyPrintErr("Error creating temporary file:\tErr: %s", err)
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
			integration.PrettyPrintWarn("Error running kubectl command '%v':\n\tResult: %s\tErr: %s", args, result, err)
		}

		lines := strings.Split(result, " ")
		if len(lines) < workerCount+1 {
			integration.PrettyPrint("Service status output too short. Waiting %v then checking again.", waitTime)
			time.Sleep(waitTime)
			waitTime *= 2
			continue
		}
		if lines[0] != "Running" || lines[1] != "Running" {
			integration.PrettyPrint("Services not running. Waiting %v then checking again.", waitTime)
			time.Sleep(waitTime)
			waitTime *= 2
		} else {
			done = true
		}
	}
	integration.PrettyPrint("")
}

func displayTestPods() {
	result, err := runKubectlCommand([]string{"--namespace=" + testNamespace, "get", "pods", "-o=wide"})
	integration.PrettyPrint("Pods are running:\n%s", result)

	if err != nil {
		integration.PrettyPrintWarn("Error running kubectl command '%v'", err)
	}

	integration.PrettyPrint("")
}

func fetchTestResults() {
	orchestratorPodName := getPodName(orchestratorName)
	sleep := 30 * time.Second

	for len(orchestratorPodName) == 0 {
		integration.PrettyPrintInfo("Waiting %s for orchestrator pod creation", sleep)
		time.Sleep(sleep)
		orchestratorPodName = getPodName(orchestratorName)
	}
	integration.PrettyPrint("Orchestrator Pod is %s", orchestratorPodName)
	sleep = 5 * time.Minute
	// The pods orchestrate themselves, we just wait for the results file to show up in the orchestrator container
	for true {
		// Monitor the orchestrator pod for the CSV results file
		csvdata := getCsvResultsFromPod(orchestratorPodName)
		if csvdata == nil {
			integration.PrettyPrintInfo("Scanned orchestrator pod filesystem - no results file found yet...waiting %s for orchestrator to write CSV file...", sleep)
			time.Sleep(sleep)
			continue
		}
		integration.PrettyPrint("Test concluded - CSV raw data written to %s", netperfOpts.Output)
		if processCsvData(orchestratorPodName, "/tmp/result.csv", "/tmp/output.txt") {
			break
		}
	}
}

// Retrieve the logs for the pod/container and check if csv data has been generated
func getCsvResultsFromPod(podName string) *string {
	logData, err := runKubectlCommand([]string{"--namespace=" + testNamespace, "logs", podName, "--timestamps=false"})
	if err != nil {
		integration.PrettyPrintWarn("Error reading logs from pod %s:\n\tResult: %s\tErr: %s", podName, logData, err)
		return nil
	}

	index := strings.Index(logData, csvDataMarker)
	endIndex := strings.Index(logData, csvEndDataMarker)
	if index == -1 || endIndex == -1 {
		return nil
	}

	csvData := string(logData[index+len(csvDataMarker)+1: endIndex])
	return &csvData
}

// processCsvData : Fetch the CSV datafile
func processCsvData(podName, remoteFile, remoteRawFile string) bool {
	remote := fmt.Sprintf("%s/%s:%s", testNamespace, podName, remoteFile)
	_, err := runKubectlCommand([]string{"cp", remote, netperfOpts.Output})

	if err != nil {
		integration.PrettyPrintErr("Couldn't copy output CSV datafile: %s", err)
		return false
	}

	remote = fmt.Sprintf("%s/%s:%s", testNamespace, podName, remoteRawFile)
	_, err = runKubectlCommand([]string{"cp", remote, fmt.Sprintf("%s.raw", netperfOpts.Output)})

	if err != nil {
		integration.PrettyPrintErr("Couldn't copy output RAW datafile: %s", err)
		return false
	}

	return true
}

func removeServices() {
	removeService(orchestratorName)
	removeService(fmt.Sprintf("%s%d", workerName, 2))
}

func removeService(name string) {
	_, err := runKubectlCommand([]string{"--namespace=" + testNamespace, "delete", "svc/" + name})

	if err != nil {
		integration.PrettyPrintWarn("Error deleting service '%v'", name, err)
	}
}

func removeReplicationControllers() {
	removeReplicationController(orchestratorName)
	for i := 1; i <= workerCount; i++ {
		removeReplicationController(fmt.Sprintf("%s%d", workerName, i))
	}
}

func removeReplicationController(name string) {
	_, err := runKubectlCommand([]string{"--namespace=" + testNamespace, "delete", "rc/" + name})

	if err != nil {
		integration.PrettyPrintWarn("Error deleting replication-controller '%v'", name, err)
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

func getServiceIP(name string) (string, error) {
	template := "\"{..spec.clusterIP}\""
	args := []string{"--namespace=" + testNamespace, "get", "service", "-l", "app=" + name, "-o", "jsonpath=" + template}
	result, err := runKubectlCommand(args)

	if err != nil {
		return "", err
	}

	return strings.Trim(result, " \n"), nil
}
