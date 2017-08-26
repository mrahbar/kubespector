package pkg

import (
	"fmt"
    "github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	node    types.Node
	sshOpts types.SSHConfig
)

const (
	netperfNamespace = "netperf"
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
var netperfCmdExecutor *ssh.CommandExecutor

func Netperf(cmdParams *types.CommandParams) {
    initParams(cmdParams)
    netperfOpts = cmdParams.Opts.(*types.NetperfOpts)
	group := util.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
        printer.PrintErr("No host configured for group [%s]", types.MASTER_GROUPNAME)
		os.Exit(1)
	}

	sshOpts = config.Ssh
    node = ssh.GetFirstAccessibleNode(sshOpts, group.Nodes, printer)

	if !util.IsNodeAddressValid(node) {
        printer.PrintErr("No master available")
		os.Exit(1)
	}

	if netperfOpts.OutputDir == "" {
		exPath, err := util.GetExecutablePath()
		if err != nil {
			os.Exit(1)
		}
		netperfOpts.OutputDir = path.Join(exPath, "netperf-results")
	}

	err := os.MkdirAll(netperfOpts.OutputDir, os.ModePerm)
	if err != nil {
        printer.PrintErr("Failed to open output file for path %s Error: %v", netperfOpts.OutputDir, err)
		os.Exit(1)
	}

    printer.Print("Running kubectl commands on node %s", util.ToNodeLabel(node))

    netperfCmdExecutor = &ssh.CommandExecutor{
        SshOpts: config.Ssh,
        Node:    node,
        Printer: printer,
    }

	checkingNetperfPreconditions()
	createNetperfNamespace()
	createNetperfServices()
	createNetperfReplicationControllers()

	waitForNetperfServicesToBeRunning()
	displayNetperfPods()
	fetchTestResults()

	if netperfOpts.Cleanup {
        printer.PrintInfo("Cleaning up...")
		removeNetperfServices()
		removeNetperfReplicationControllers()
	}

    printer.PrintOk("DONE")
}

func checkingNetperfPreconditions() {
    count, err := netperfCmdExecutor.GetNumberOfReadyNodes()

	if err != nil {
        printer.PrintErr("Error checking node count: %s", err)
		os.Exit(1)
	} else if count < 2 {
        printer.PrintErr("Insufficient number of nodes for netperf test (need minimum of 2 nodes)")
		os.Exit(1)
	}
}

func createNetperfNamespace() {
    printer.PrintInfo("Creating namespace")
    err := netperfCmdExecutor.CreateNamespace(netperfNamespace)

	if err != nil {
        printer.PrintErr("Error creating test namespace: %s", err)
		os.Exit(1)
	} else {
        printer.PrintOk("Namespace %s created", netperfNamespace)
    }
    integration.PrettyNewLine()
}

func createNetperfServices() {
    printer.PrintInfo("Creating services")
	// Host
	data := types.Service{Name: orchestratorName, Namespace: netperfNamespace, Ports: []types.ServicePort{
		{
			Name:       orchestratorName,
			Port:       orchestratorPort,
			Protocol:   "TCP",
			TargetPort: orchestratorPort,
		},
	}}
    exists, err := netperfCmdExecutor.CreateService(data)
	if exists {
        printer.PrintIgnored("Service: %s already exists.", orchestratorName)
	} else {
        printer.PrintErr("Error adding service %v: %s", orchestratorName, err)
		os.Exit(1)
	}

	// Create the netperf-w2 service that points a clusterIP at the worker 2 pod
	name := fmt.Sprintf("%s%d", workerName, 2)
	data = types.Service{Name: name, Namespace: netperfNamespace, Ports: []types.ServicePort{
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
    exists, err = netperfCmdExecutor.CreateService(data)
	if exists {
        printer.PrintIgnored("Service: %s already exists.", name)
	} else {
        printer.PrintErr("Error adding service %v: %s", name, err)
		os.Exit(1)
	}
    integration.PrettyNewLine()
}

func createNetperfReplicationControllers() {
    printer.PrintInfo("Creating ReplicationControllers")

	hostRC := types.ReplicationController{Name: orchestratorName, Namespace: netperfNamespace,
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
    err := netperfCmdExecutor.CreateReplicationController(hostRC)

	if err != nil {
        printer.PrintErr("Error creating %s replication controller: %s", orchestratorName, err)
		os.Exit(1)
	} else {
        printer.PrintOk("Created %s replication-controller", orchestratorName)
	}

	args := []string{"get", "nodes", " | ", "grep", "-w", "\"Ready\"", " | ", "sed", "-e", "\"s/[[:space:]]\\+/,/g\""}
    sshOut, err := netperfCmdExecutor.RunKubectlCommand(args)

	if err != nil {
        printer.PrintErr("Error getting nodes for worker replication controller: %s", err)
		os.Exit(1)
	} else {
        printer.Print("Waiting 5s to give orchestrator pod time to start")
		time.Sleep(5 * time.Second)
		hostIP, err := getServiceIP(orchestratorName)
		if hostIP == "" || err != nil {
            printer.PrintErr("Error getting clusterIP of service %s: %s", orchestratorName, err)
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

			clientRC := types.ReplicationController{Name: name, Namespace: netperfNamespace, Image: netperfImage,
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

            _, err := netperfCmdExecutor.DeployKubernetesResource(types.REPLICATION_CONTROLLER_TEMPLATE, clientRC)

			if err != nil {
                printer.PrintErr("Error creating %s replication controller: %s", name, err)
				os.Exit(1)
			} else {
                printer.PrintOk("Created %s replication-controller", name)
			}
		}
	}
    integration.PrettyNewLine()
}

func waitForNetperfServicesToBeRunning() {
    printer.PrintInfo("Waiting for pods to be Running...")
	waitTime := time.Second
	done := false
	for !done {
		tmpl := "\"{..status.phase}\""
		args := []string{"--namespace=" + netperfNamespace, "get", "pods", "-o", "jsonpath=" + tmpl}
        sshOut, err := netperfCmdExecutor.RunKubectlCommand(args)

		if err != nil {
            printer.PrintWarn("Error running kubectl command '%v': %s", args, err)
		}

		lines := strings.Split(sshOut.Stdout, " ")
		if len(lines) < workerCount+1 {
            printer.Print("Service status output too short. Waiting %v then checking again.", waitTime)
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
            printer.Print("Services not running. Waiting %v then checking again.", waitTime)
			time.Sleep(waitTime)
			waitTime *= 2
		} else {
			done = true
		}
	}
    integration.PrettyNewLine()
}

func displayNetperfPods() {
    result, err := netperfCmdExecutor.GetPods(netperfNamespace, true)
	if err != nil {
        printer.PrintWarn("Error running kubectl command '%v'", err)
	} else {
        printer.Print("Pods are running\n%s", result)
    }

    integration.PrettyNewLine()
}

func fetchTestResults() {
    printer.PrintInfo("Waiting till pods orchestrate themselves. This may take several minutes..")
	orchestratorPodName := getPodName(orchestratorName)
	sleep := 30 * time.Second

	for len(orchestratorPodName) == 0 {
        printer.PrintInfo("Waiting %s for orchestrator pod creation", sleep)
		time.Sleep(sleep)
		orchestratorPodName = getPodName(orchestratorName)
	}
    printer.Print("The pods orchestrate themselves, waiting for the results file to show up in the orchestrator pod %s", orchestratorPodName)
	sleep = 5 * time.Minute
    integration.PrettyNewLine()

	for true {
		// Monitor the orchestrator pod for the CSV results file
		csvdata := getCsvResultsFromPod(orchestratorPodName)
		if csvdata == nil {
            printer.PrintSkipped("Scanned orchestrator pod filesystem - no results file found yet...waiting %s for orchestrator to write CSV file...", sleep)
			time.Sleep(sleep)
			continue
		}
        printer.PrintInfo("Test concluded - CSV raw data written to %s", netperfOpts.OutputDir)
		if processCsvData(orchestratorPodName) {
			break
		}
	}
}

// Retrieve the logs for the pod/container and check if csv data has been generated
func getCsvResultsFromPod(podName string) *string {
	args := []string{"--namespace=" + netperfNamespace, "logs", podName, "--timestamps=false"}
    sshOut, err := netperfCmdExecutor.RunKubectlCommand(args)
	logData := sshOut.Stdout
	if err != nil {
        printer.PrintWarn("Error reading logs from pod %s: %s", podName, err)
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
	remote := fmt.Sprintf("%s/%s:%s", netperfNamespace, podName, resultCaptureFile)
    _, err := netperfCmdExecutor.RunKubectlCommand([]string{"cp", remote, resultCaptureFile})
	if err != nil {
        printer.PrintErr("Couldn't copy output CSV datafile %s from remote %s: %s",
			resultCaptureFile, util.GetNodeAddress(node), err)
		return false
	}

    err = netperfCmdExecutor.DownloadFile(resultCaptureFile, filepath.Join(netperfOpts.OutputDir, "result.csv"))
	if err != nil {
        printer.PrintErr("Couldn't fetch output CSV datafile %s from remote %s: %s",
			resultCaptureFile, util.GetNodeAddress(node), err)
		return false
	}

	remote = fmt.Sprintf("%s/%s:%s", netperfNamespace, podName, outputCaptureFile)
    _, err = netperfCmdExecutor.RunKubectlCommand([]string{"cp", remote, outputCaptureFile})
	if err != nil {
        printer.PrintErr("Couldn't copy output RAW datafile %s from remote %s: %s",
			outputCaptureFile, util.GetNodeAddress(node), err)
		return false
	}
    err = netperfCmdExecutor.DownloadFile(outputCaptureFile, filepath.Join(netperfOpts.OutputDir, "output.txt"))
	if err != nil {
        printer.PrintErr("Couldn't fetch output RAW datafile %s from remote %s: %s",
			outputCaptureFile, util.GetNodeAddress(node), err)
		return false
	}

	return true
}

func removeNetperfServices() {
	name := "svc/" + orchestratorName
    err := netperfCmdExecutor.RemoveResource(netperfNamespace, name)
	if err != nil {
        printer.PrintWarn("Error deleting service '%v'", name, err)
	}

	name = fmt.Sprintf("svc/%s%d", workerName, 2)
    err = netperfCmdExecutor.RemoveResource(netperfNamespace, name)
	if err != nil {
        printer.PrintWarn("Error deleting service '%v'", name, err)
	}
}

func removeNetperfReplicationControllers() {
    err := netperfCmdExecutor.RemoveResource(netperfNamespace, orchestratorName)
	if err != nil {
        printer.PrintWarn("Error deleting replication-controller '%v'", orchestratorName, err)
	}

	for i := 1; i <= workerCount; i++ {
		name := fmt.Sprintf("rc/%s%d", workerName, i)
        err := netperfCmdExecutor.RemoveResource(netperfNamespace, name)
		if err != nil {
            printer.PrintWarn("Error deleting replication-controller '%v'", name, err)
		}
	}
}

func getPodName(name string) string {
	tmpl := "\"{..metadata.name}\""
	args := []string{"--namespace=" + netperfNamespace, "get", "pods", "-l", "app=" + name, "-o", "jsonpath=" + tmpl}
    sshOut, err := netperfCmdExecutor.RunKubectlCommand(args)

	if err != nil {
		return ""
	}

	return strings.TrimRight(sshOut.Stdout, "\n")
}

func getServiceIP(name string) (string, error) {
	tmpl := "\"{..spec.clusterIP}\""
	args := []string{"--namespace=" + netperfNamespace, "get", "service", "-l", "app=" + name, "-o", "jsonpath=" + tmpl}
    sshOut, err := netperfCmdExecutor.RunKubectlCommand(args)

	if err != nil {
		return "", err
	}

	return strings.Trim(sshOut.Stdout, " \n"), nil
}
