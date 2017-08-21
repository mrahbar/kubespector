package pkg

import (
	"bytes"
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
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
	node    types.Node
	sshOpts types.SSHConfig
)

const (
	testNamespace    = "netperf"
	orchestratorName = "netperf-orchestrator"
	workerName       = "netperf-w"

	orchestratorMode = "orchestrator"
	workerMode       = "worker"

	csvDataMarker     = "GENERATING CSV OUTPUT"
	csvEndDataMarker  = "END CSV DATA"
	outputCaptureFile = "/tmp/output.txt"
	resultCaptureFile = "/tmp/result.csv"

	netperfImage = "endianogino/netperf:1.1"

	workerCount      = 3
	orchestratorPort = 5202
	iperf3Port       = 5201
	netperfPort      = 12865
)

var netperfOpts *types.NetperfOpts

func Netperf(config types.Config, opts *types.NetperfOpts) {
	netperfOpts = opts
	group := util.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		util.PrettyPrintErr("No host configured for group [%s]", types.MASTER_GROUPNAME)
		os.Exit(1)
	}

	sshOpts = config.Ssh
	node = ssh.GetFirstAccessibleNode(sshOpts, group.Nodes, netperfOpts.Debug)

	if !util.IsNodeAddressValid(node) {
		util.PrettyPrintErr("No master available")
		os.Exit(1)
	}

	if netperfOpts.OutputDir == "" {
		ex, err := os.Executable()
		if err != nil {
			os.Exit(1)
		}
		exPath := path.Dir(ex)
		netperfOpts.OutputDir = path.Join(exPath, "netperf-results")
	}

	err := os.MkdirAll(netperfOpts.OutputDir, os.ModePerm)
	if err != nil {
		util.PrettyPrintErr("Failed to open output file for path %s Error: %v", netperfOpts.OutputDir, err)
		os.Exit(1)
	}

	util.PrettyPrint("Running kubectl commands on node %s", util.ToNodeLabel(node))

	checkingPreconditions()
	createTestNamespace()
	createServices()
	createReplicationControllers()

	waitForServicesToBeRunning()
	displayTestPods()
	fetchTestResults()

	// cleanup services
	if netperfOpts.Cleanup {
		util.PrettyPrintInfo("Cleaning up...")
		removeServices()
		removeReplicationControllers()
	}

	util.PrettyPrintOk("DONE")
}

func checkingPreconditions() {
	tmpl := "\"{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}\""
	args := []string{"get", "nodes", "-o", "jsonpath=" + tmpl, " | ", "tr", "';'", "\\\\n", " | ", "grep", "\"Ready=True\"", " | ", "wc", "-l"}
	sshOut, err := runKubectlCommand(args)

	if err != nil {
		util.PrettyPrintErr("Error checking node count: %s", err)
		os.Exit(1)
	} else {
		count, errAtoi := strconv.Atoi(strings.TrimRight(sshOut.Stdout, "\n"))

		if errAtoi != nil {
			util.PrettyPrintErr("Error getting node count: %s", errAtoi)
			os.Exit(1)
		} else if count < 2 {
			util.PrettyPrintErr("Insufficient number of nodes for netperf test (need minimum 2 nodes)")
			os.Exit(1)
		}
	}
}

func createTestNamespace() {
	util.PrettyPrintInfo("Creating namespace")
	data := make(map[string]string)
	data["Namespace"] = testNamespace
	_, err := deployKubernetesResource(types.NAMESPACE_TEMPLATE, data)

	if err != nil {
		util.PrettyPrintErr("Error creating test namespace: %s", err)
		os.Exit(1)
	} else {
		util.PrettyPrintOk("Namespace %s created", testNamespace)
	}
	util.PrettyNewLine()
}

func createServices() {
	util.PrettyPrintInfo("Creating services")
	// Host
	data := types.Service{Name: orchestratorName, Namespace: testNamespace, Ports: []types.ServicePort{
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
	data = types.Service{Name: name, Namespace: testNamespace, Ports: []types.ServicePort{
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
	util.PrettyNewLine()
}

func createService(name string, serviceData interface{}) {
	sshOut, err := deployKubernetesResource(types.SERVICE_TEMPLATE, serviceData)

	if err != nil {
		if strings.Contains(ssh.CombineOutput(sshOut), "AlreadyExists") {
			util.PrettyPrintIgnored("Service: %s already exists.", name)
		} else {
			util.PrettyPrintErr("Error adding service %v: %s", name, err)
			os.Exit(1)
		}
	} else {
		util.PrettyPrintOk("Service %s created.", name)
	}
}

func createReplicationControllers() {
	util.PrettyPrintInfo("Creating ReplicationControllers")

	hostRC := types.ReplicationController{Name: orchestratorName, Namespace: testNamespace,
		Image: netperfImage,
		Args: []types.Arg{
			{
				Key:   "--mode",
				Value: orchestratorMode,
			},
		},
		Ports: []types.PodPort{
			{
				Name:     "service-port",
				Protocol: "TCP",
				Port:     orchestratorPort,
			},
		},
	}
	sshOut, err := deployKubernetesResource(types.REPLICATION_CONTROLLER_TEMPLATE, hostRC)

	if err != nil {
		util.PrettyPrintErr("Error creating %s replication controller: %s", orchestratorName, err)
		os.Exit(1)
	} else {
		util.PrettyPrintOk("Created %s replication-controller", orchestratorName)
	}

	args := []string{"get", "nodes", " | ", "grep", "-w", "\"Ready\"", " | ", "sed", "-e", "\"s/[[:space:]]\\+/,/g\""}
	sshOut, err = runKubectlCommand(args)

	if err != nil {
		util.PrettyPrintErr("Error getting nodes for worker replication controller: %s", err)
		os.Exit(1)
	} else {
		util.PrettyPrint("Waiting 5s to give orchestrator pod time to start")
		time.Sleep(5 * time.Second)
		hostIP, err := getServiceIP(orchestratorName)
		if hostIP == "" || err != nil {
			util.PrettyPrintErr("Error getting clusterIP of service %s: %s", orchestratorName, err)
			os.Exit(1)
		}

		lines := strings.SplitN(sshOut.Stdout, "\n", -1)
		firstNode := strings.Split(lines[0], ",")[0]
		secondNode := strings.Split(lines[1], ",")[0]

		for i := 1; i <= workerCount; i++ {
			name := fmt.Sprintf("%s%d", workerName, i)
			kubeNode := firstNode
			if i == 3 {
				kubeNode = secondNode
			}

			clientRC := types.ReplicationController{Name: name, Namespace: testNamespace, Image: netperfImage,
				NodeName: kubeNode,
				Args: []types.Arg{
					{
						Key:   "--mode",
						Value: workerMode,
					},
				},
				Ports: []types.PodPort{
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
				Envs: []types.Env{
					{
						Name:  "workerName",
						Value: name,
					},
					{
						Name:       "workerPodIP",
						FieldValue: "status.podIP",
					},
					{
						Name:  "orchestratorPort",
						Value: "5202",
					},
					{
						Name:  "orchestratorPodIP",
						Value: hostIP,
					},
				},
			}

			_, err := deployKubernetesResource(types.REPLICATION_CONTROLLER_TEMPLATE, clientRC)

			if err != nil {
				util.PrettyPrintErr("Error creating %s replication controller: %s", name, err)
				os.Exit(1)
			} else {
				util.PrettyPrintOk("Created %s replication-controller", name)
			}
		}
	}
	util.PrettyNewLine()
}

func runKubectlCommand(args []string) (*types.SSHOutput, error) {
	a := strings.Join(args, " ")
	return ssh.PerformCmd(sshOpts, node, fmt.Sprintf("kubectl %s", a), netperfOpts.Debug)
}

func deployKubernetesResource(tpl string, data interface{}) (*types.SSHOutput, error) {
	var definition bytes.Buffer

	tmpl, _ := template.New("kube-template").Parse(tpl)
	tmpl.Execute(&definition, data)

	tmpFile, err := ioutil.TempFile("", "kubeceptor-")
	if err != nil {
		util.PrettyPrintErr("Error creating temporary file: %s", err)
		os.Exit(1)
	}

	defer os.Remove(tmpFile.Name())
	ioutil.WriteFile(tmpFile.Name(), definition.Bytes(), os.ModeAppend)
	remoteFile := path.Join("/tmp", filepath.Base(tmpFile.Name()))
	err = ssh.UploadFile(sshOpts, node, remoteFile, tmpFile.Name(), netperfOpts.Debug)
	if err != nil {
		util.PrettyPrintErr("Error transferring temporary file %s: %s", tmpFile.Name(), err)
		os.Exit(1)
	}

	args := []string{"apply", "-f", remoteFile}
	result, err := runKubectlCommand(args)
	ssh.DeleteRemoteFile(sshOpts, node, remoteFile, netperfOpts.Debug)

	return result, err
}

func waitForServicesToBeRunning() {
	util.PrettyPrintInfo("Waiting for pods to be Running...")
	waitTime := time.Second
	done := false
	for !done {
		tmpl := "\"{..status.phase}\""
		args := []string{"--namespace=" + testNamespace, "get", "pods", "-o", "jsonpath=" + tmpl}
		sshOut, err := runKubectlCommand(args)

		if err != nil {
			util.PrettyPrintWarn("Error running kubectl command '%v': %s", args, err)
		}

		lines := strings.Split(sshOut.Stdout, " ")
		if len(lines) < workerCount+1 {
			util.PrettyPrint("Service status output too short. Waiting %v then checking again.", waitTime)
			time.Sleep(waitTime)
			waitTime *= 2
			continue
		}

		allRunning := true
		for _, p := range lines {
			if p != "Running" {
				allRunning = false
				break
			}
		}
		if !allRunning {
			util.PrettyPrint("Services not running. Waiting %v then checking again.", waitTime)
			time.Sleep(waitTime)
			waitTime *= 2
		} else {
			done = true
		}
	}
	util.PrettyNewLine()
}

func displayTestPods() {
	result, err := runKubectlCommand([]string{"--namespace=" + testNamespace, "get", "pods", "-o=wide"})
	util.PrettyPrint("Pods are running\n%s", result)

	if err != nil {
		util.PrettyPrintWarn("Error running kubectl command '%v'", err)
	}

	util.PrettyNewLine()
}

func fetchTestResults() {
	util.PrettyPrintInfo("Waiting till pods orchestrate themselves. This may take several minutes..")
	orchestratorPodName := getPodName(orchestratorName)
	sleep := 30 * time.Second

	for len(orchestratorPodName) == 0 {
		util.PrettyPrintInfo("Waiting %s for orchestrator pod creation", sleep)
		time.Sleep(sleep)
		orchestratorPodName = getPodName(orchestratorName)
	}
	util.PrettyPrint("The pods orchestrate themselves, waiting for the results file to show up in the orchestrator pod %s", orchestratorPodName)
	sleep = 5 * time.Minute
	util.PrettyNewLine()

	for true {
		// Monitor the orchestrator pod for the CSV results file
		csvdata := getCsvResultsFromPod(orchestratorPodName)
		if csvdata == nil {
			util.PrettyPrintSkipped("Scanned orchestrator pod filesystem - no results file found yet...waiting %s for orchestrator to write CSV file...", sleep)
			time.Sleep(sleep)
			continue
		}
		util.PrettyPrintInfo("Test concluded - CSV raw data written to %s", netperfOpts.OutputDir)
		if processCsvData(orchestratorPodName) {
			break
		}
	}
}

// Retrieve the logs for the pod/container and check if csv data has been generated
func getCsvResultsFromPod(podName string) *string {
	sshOut, err := runKubectlCommand([]string{"--namespace=" + testNamespace, "logs", podName, "--timestamps=false"})
	logData := sshOut.Stdout
	if err != nil {
		util.PrettyPrintWarn("Error reading logs from pod %s: %s", podName, err)
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
func processCsvData(podName string) bool {
	remote := fmt.Sprintf("%s/%s:%s", testNamespace, podName, resultCaptureFile)
	_, err := runKubectlCommand([]string{"cp", remote, resultCaptureFile})
	if err != nil {
		util.PrettyPrintErr("Couldn't copy output CSV datafile %s from remote %s: %s",
			resultCaptureFile, util.GetNodeAddress(node), err)
		return false
	}

	err = ssh.DownloadFile(sshOpts, node, resultCaptureFile, filepath.Join(netperfOpts.OutputDir, "result.csv"), netperfOpts.Debug)
	if err != nil {
		util.PrettyPrintErr("Couldn't fetch output CSV datafile %s from remote %s: %s",
			resultCaptureFile, util.GetNodeAddress(node), err)
		return false
	}

	remote = fmt.Sprintf("%s/%s:%s", testNamespace, podName, outputCaptureFile)
	_, err = runKubectlCommand([]string{"cp", remote, outputCaptureFile})
	if err != nil {
		util.PrettyPrintErr("Couldn't copy output RAW datafile %s from remote %s: %s",
			outputCaptureFile, util.GetNodeAddress(node), err)
		return false
	}
	err = ssh.DownloadFile(sshOpts, node, outputCaptureFile, filepath.Join(netperfOpts.OutputDir, "output.txt"), netperfOpts.Debug)
	if err != nil {
		util.PrettyPrintErr("Couldn't fetch output RAW datafile %s from remote %s: %s",
			outputCaptureFile, util.GetNodeAddress(node), err)
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
		util.PrettyPrintWarn("Error deleting service '%v'", name, err)
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
		util.PrettyPrintWarn("Error deleting replication-controller '%v'", name, err)
	}
}

func getPodName(name string) string {
	tmpl := "\"{..metadata.name}\""
	args := []string{"--namespace=" + testNamespace, "get", "pods", "-l", "app=" + name, "-o", "jsonpath=" + tmpl}
	sshOut, err := runKubectlCommand(args)

	if err != nil {
		return ""
	}

	return strings.TrimRight(sshOut.Stdout, "\n")
}

func getServiceIP(name string) (string, error) {
	tmpl := "\"{..spec.clusterIP}\""
	args := []string{"--namespace=" + testNamespace, "get", "service", "-l", "app=" + name, "-o", "jsonpath=" + tmpl}
	sshOut, err := runKubectlCommand(args)

	if err != nil {
		return "", err
	}

	return strings.Trim(sshOut.Stdout, " \n"), nil
}
