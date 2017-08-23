package pkg

import (
	"github.com/mrahbar/kubernetes-inspector/types"
	"os"
	"github.com/mrahbar/kubernetes-inspector/util"
	"github.com/mrahbar/kubernetes-inspector/ssh"
	"path"
	"time"
	"strings"
	"fmt"
	"io/ioutil"
	"net/http"
	"github.com/streadway/quantile"
	"encoding/json"
	"sync"
)

const (
	scaleTestNamespace = "scaletest"

	vegetaImage = "gcr.io/google_containers/loader:0.6"
	nginxImage  = "endianogino/simple-webserver:1.0"

	vegetaName = "vegeta"
	nginxName  = "nginx"

	nginxPort = 80
)

type (
	// Metrics holds metrics computed out of a slice of Results which are used
	// in some of the Reporters
	loadbotMetrics struct {
		// Latencies holds computed request latency metrics.
		Latencies latencyMetrics `json:"latencies"`
		// BytesIn holds computed incoming byte metrics.
		BytesIn byteMetrics `json:"bytes_in"`
		// BytesOut holds computed outgoing byte metrics.
		BytesOut byteMetrics `json:"bytes_out"`
		// First is the earliest timestamp in a Result set.
		Earliest time.Time `json:"earliest"`
		// Latest is the latest timestamp in a Result set.
		Latest time.Time `json:"latest"`
		// End is the latest timestamp in a Result set plus its latency.
		End time.Time `json:"end"`
		// Duration is the duration of the attack.
		Duration time.Duration `json:"duration"`
		// Wait is the extra time waiting for responses from targets.
		Wait time.Duration `json:"wait"`
		// Requests is the total number of requests executed.
		Requests uint64 `json:"requests"`
		// Rate is the rate of requests per second.
		Rate float64 `json:"rate"`
		// Success is the percentage of non-error responses.
		Success float64 `json:"success"`
		// StatusCodes is a histogram of the responses' status codes.
		StatusCodes map[string]int `json:"status_codes"`
		// Errors is a set of unique errors returned by the targets during the attack.
		Errors []string `json:"errors"`

		errors    map[string]struct{}
		success   uint64
		latencies *quantile.Estimator
	}

	// LatencyMetrics holds computed request latency metrics.
	latencyMetrics struct {
		// Total is the total latency sum of all requests in an attack.
		Total time.Duration `json:"total"`
		// Mean is the mean request latency.
		Mean time.Duration `json:"mean"`
		// P50 is the 50th percentile request latency.
		P50 time.Duration `json:"50th"`
		// P95 is the 95th percentile request latency.
		P95 time.Duration `json:"95th"`
		// P99 is the 99th percentile request latency.
		P99 time.Duration `json:"99th"`
		// Max is the maximum observed request latency.
		Max time.Duration `json:"max"`
	}

	// ByteMetrics holds computed byte flow metrics.
	byteMetrics struct {
		// Total is the total number of flowing bytes in an attack.
		Total uint64 `json:"total"`
		// Mean is the mean number of flowing bytes per hit.
		Mean float64 `json:"mean"`
	}
)

var scaleTestOpts *types.ScaleTestOpts

func ScaleTest(config types.Config, opts *types.ScaleTestOpts) {
	scaleTestOpts = opts
	group := util.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)

	if group.Nodes == nil || len(group.Nodes) == 0 {
		util.PrettyPrintErr("No host configured for group [%s]", types.MASTER_GROUPNAME)
		os.Exit(1)
	}

	sshOpts = config.Ssh
	node = ssh.GetFirstAccessibleNode(sshOpts, group.Nodes, scaleTestOpts.Debug)

	if !util.IsNodeAddressValid(node) {
		util.PrettyPrintErr("No master available")
		os.Exit(1)
	}

	if scaleTestOpts.OutputDir == "" {
		exPath, err := util.GetExecutablePath()
		if err != nil {
			os.Exit(1)
		}
		scaleTestOpts.OutputDir = path.Join(exPath, "scaletest-results")
	}

	err := os.MkdirAll(scaleTestOpts.OutputDir, os.ModePerm)
	if err != nil {
		util.PrettyPrintErr("Failed to open output file for path %s Error: %v", scaleTestOpts.OutputDir, err)
		os.Exit(1)
	}

	util.PrettyPrint("Running kubectl commands on node %s", util.ToNodeLabel(node))

	checkingScaleTestPreconditions()
	createScaleTestNamespace()
	createScaleTestServices()
	createScaleTestReplicationControllers()

	waitForScaleTestServicesToBeRunning()
	displayScaleTestPods()

	//TODO scale scenario: run  nginx - vegeta
	// 1-1 (idle)
	// 10-1 (under-load)
	// 10-10 (equal-load)
	// 10-100 (overload-load)
	// 100-1000 (one million requests per second)
	// scale/wait, query QPS, iterate

	if scaleTestOpts.Cleanup {
		util.PrettyPrintInfo("Cleaning up...")
		removeScaleTest()
	}

	util.PrettyPrintOk("DONE")
}

func checkingScaleTestPreconditions() {
	count, err := ssh.GetNumberOfReadyNodes(sshOpts, node, scaleTestOpts.Debug)

	if err != nil {
		util.PrettyPrintErr("Error checking node count: %s", err)
		os.Exit(1)
	} else if count < 1 {
		util.PrettyPrintErr("Insufficient number of nodes for netperf test (need minimum of 1 node)")
		os.Exit(1)
	}
}

func createScaleTestNamespace() {
	util.PrettyPrintInfo("Creating namespace")
	err := ssh.CreateNamespace(sshOpts, node, scaleTestNamespace, scaleTestOpts.Debug)

	if err != nil {
		util.PrettyPrintErr("Error creating test namespace: %s", err)
		os.Exit(1)
	} else {
		util.PrettyPrintOk("Namespace %s created", scaleTestNamespace)
	}
	util.PrettyNewLine()
}

func createScaleTestServices() {
	util.PrettyPrintInfo("Creating services")

	data := types.Service{Name: nginxName, Namespace: scaleTestNamespace, Ports: []types.ServicePort{
		{
			Name:       "http-port",
			Port:       nginxPort,
			Protocol:   "TCP",
			TargetPort: nginxPort,
		},
	}}

	exists, err := ssh.CreateService(sshOpts, node, data, scaleTestOpts.Debug)
	if exists {
		util.PrettyPrintIgnored("Service: %s already exists.", nginxName)
	} else {
		util.PrettyPrintErr("Error adding service %v: %s", nginxName, err)
		os.Exit(1)
	}

	util.PrettyNewLine()
}

func createScaleTestReplicationControllers() {
	util.PrettyPrintInfo("Creating ReplicationControllers")

	vegetaRC := types.ReplicationController{Name: vegetaName, Namespace: scaleTestNamespace, Image: vegetaImage,
		Commands: []string{"/loader", "-host=nginx", "-rate=1000", "-address=:8080", "-workers=10", "-duration=1s"},
		ResourceRequest: types.ResourceRequest{Cpu: "100m"},
	}

	nginxRc := types.ReplicationController{Name: nginxName, Namespace: scaleTestNamespace, Image: nginxImage,
		ResourceRequest: types.ResourceRequest{Cpu: "1000m"},
	}

	err := ssh.CreateReplicationController(sshOpts, node, vegetaRC, scaleTestOpts.Debug)
	if err != nil {
		util.PrettyPrintErr("Error creating %s replication controller: %s", vegetaName, err)
		os.Exit(1)
	} else {
		util.PrettyPrintOk("Created %s replication-controller", vegetaName)
	}

	err = ssh.CreateReplicationController(sshOpts, node, nginxRc, scaleTestOpts.Debug)
	if err != nil {
		util.PrettyPrintErr("Error creating %s replication controller: %s", nginxName, err)
		os.Exit(1)
	} else {
		util.PrettyPrintOk("Created %s replication-controller", nginxName)
	}
}

func waitForScaleTestServicesToBeRunning() {
	util.PrettyPrintInfo("Waiting for pods to be Running...")
	waitTime := time.Second
	done := false
	for !done {
		tmpl := "\"{..status.phase}\""
		args := []string{"--namespace=" + scaleTestNamespace, "get", "pods", "-o", "jsonpath=" + tmpl}
		sshOut, err := ssh.RunKubectlCommand(sshOpts, node, args, netperfOpts.Debug)

		if err != nil {
			util.PrettyPrintWarn("Error running kubectl command '%v': %s", args, err)
		}

		lines := strings.Split(sshOut.Stdout, " ")
		if len(lines) < 2 {
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

func displayScaleTestPods() {
	result, err := ssh.GetPods(sshOpts, node, scaleTestNamespace, true, netperfOpts.Debug)
	if err != nil {
		util.PrettyPrintWarn("Error running kubectl command '%v'", err)
	} else {
		util.PrettyPrint("Pods are running\n%s", result)
	}

	util.PrettyNewLine()
}

func fetchResults() {
	var ips []string
	var err error
	attempts := 0

	for {
		ips, err = getLoadbotPodIPs()
		if err != nil {
			if scaleTestOpts.Debug {
				util.PrettyPrintDebug("Could not get loadbot ips: %s", err)
			}
			attempts += 1
			if attempts < 3 {
				time.Sleep(2 * time.Second)
				continue
			} else {
				util.PrettyPrintErr("Failed to get loadbot ips after 3 attempts: %v", err)
				os.Exit(1)
			}
		} else {
			break
		}
	}

	parts := []loadbotMetrics{}
	lock := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(ips))
	for _, ip := range ips {
		go func(ip string) {
			defer wg.Done()
			url := "http://" + ip + ":8080/"
			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("Error getting: %v\n", err)
				return
			}
			defer resp.Body.Close()
			var data []byte
			if data, err = ioutil.ReadAll(resp.Body); err != nil {
				fmt.Printf("Error reading: %v\n", err)
				return
			}
			var metrics loadbotMetrics
			if err := json.Unmarshal(data, &metrics); err != nil {
				fmt.Printf("Error decoding: %v\n", err)
				return
			}
			lock.Lock()
			defer lock.Unlock()
			parts = append(parts, metrics)
		}(ip)
	}
	wg.Wait()
	evaluateData(parts)
	fmt.Printf("Updated.\n")
}

func evaluateData(metrics []loadbotMetrics) {
	/*
	ScaleApp.prototype.getQPS = function() {
    if (!this.fullData) {
	return 0;
    }
    var qps = 0;
    angular.forEach(this.fullData, function(value) {
	    if (value && value.rate) {
		qps += value.rate;
	    }
	});
    return qps;
};

ScaleApp.prototype.getSuccess = function() {
    if (!this.fullData) {
	return 0;
    }
    var success = 0;
    var count = 0;
    angular.forEach(this.fullData, function(value) {
	    if (value && value.success) {
		success += value.success * 100;
		count++;
	    }
	});
    return success / count;
};

ScaleApp.prototype.getLatency = function() {
    if (!this.fullData) {
	return {};
    }
    var latency = {
	"mean": 0,
	"99th": 0
    };
    var count = 0;
    angular.forEach(this.fullData, function(datum) {
	    if (datum.latencies) {
		latency.mean += datum.latencies.mean / 1000000;
		latency["99th"] += datum.latencies["99th"] / 1000000;
		count++;
	    }
	});
    if (count == 0) {
	return {};
    }
    latency.mean = (latency.mean/count);
    latency["99th"] = (latency["99th"]/count);

    return latency;
};
*/
}

func getLoadbotPodIPs() ([]string, error) {
	tmpl := "\"{..status.podIP}\""
	args := []string{"--namespace=" + scaleTestNamespace, "get", "pods", "-l", "app=" + vegetaName, "-o", "jsonpath=" + tmpl}
	sshOut, err := ssh.RunKubectlCommand(sshOpts, node, args, netperfOpts.Debug)

	if err != nil {
		return []string{}, err
	}

	return strings.Split(sshOut.Stdout, " "), nil
}

func removeScaleTest() {
	name := "svc/" + nginxName
	err := ssh.RemoveResource(sshOpts, node, scaleTestNamespace, name, netperfOpts.Debug)
	if err != nil {
		util.PrettyPrintWarn("Error deleting service '%v'", name, err)
	}

	err = ssh.RemoveResource(sshOpts, node, scaleTestNamespace, vegetaName, netperfOpts.Debug)
	if err != nil {
		util.PrettyPrintWarn("Error deleting replication-controller '%v'", vegetaName, err)
	}

	err = ssh.RemoveResource(sshOpts, node, scaleTestNamespace, nginxName, netperfOpts.Debug)
	if err != nil {
		util.PrettyPrintWarn("Error deleting replication-controller '%v'", nginxName, err)
	}
}
