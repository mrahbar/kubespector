package pkg

import (
    "encoding/json"
    "github.com/mrahbar/kubernetes-inspector/integration"
    "github.com/mrahbar/kubernetes-inspector/ssh"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
    "github.com/streadway/quantile"
    "os"
	"path"
	"strings"
	"sync"
    "time"
)

const (
	scaleTestNamespace = "scaletest"

    vegetaImage    = "gcr.io/google_containers/loader:0.6"
    webserverImage = "endianogino/simple-webserver:1.0"

    vegetaName    = "vegeta"
    webserverName = "webserver"

    webserverPort = 80
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

func ScaleTest(cmdParams *types.CommandContext) {
    initParams(cmdParams)
    scaleTestOpts = cmdParams.Opts.(*types.ScaleTestOpts)
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

	if scaleTestOpts.OutputDir == "" {
		exPath, err := util.GetExecutablePath()
		if err != nil {
			os.Exit(1)
		}
		scaleTestOpts.OutputDir = path.Join(exPath, "scaletest-results")
	}

	err := os.MkdirAll(scaleTestOpts.OutputDir, os.ModePerm)
	if err != nil {
        printer.PrintErr("Failed to open output file for path %s Error: %v", scaleTestOpts.OutputDir, err)
		os.Exit(1)
	}

    printer.Print("Running kubectl commands on node %s", util.ToNodeLabel(node))
    cmdExecutor.SetNode(node)

	checkingScaleTestPreconditions()
	createScaleTestNamespace()
	createScaleTestServices()
	createScaleTestReplicationControllers()

	waitForScaleTestServicesToBeRunning()
	displayScaleTestPods()

    runScaleTest()

	if scaleTestOpts.Cleanup {
        printer.PrintInfo("Cleaning up...")
		removeScaleTest()
	}

    printer.PrintOk("DONE")
}

func checkingScaleTestPreconditions() {
    count, err := cmdExecutor.GetNumberOfReadyNodes()

	if err != nil {
        printer.PrintErr("Error checking node count: %s", err)
		os.Exit(1)
	} else if count < 1 {
        printer.PrintErr("Insufficient number of nodes for scale test (need minimum of 1 node)")
		os.Exit(1)
	}
}

func createScaleTestNamespace() {
    printer.PrintInfo("Creating namespace")
    err := cmdExecutor.CreateNamespace(scaleTestNamespace)

	if err != nil {
        printer.PrintErr("Error creating test namespace: %s", err)
		os.Exit(1)
	} else {
        printer.PrintOk("Namespace %s created", scaleTestNamespace)
    }
    integration.PrettyNewLine()
}

func createScaleTestServices() {
    printer.PrintInfo("Creating service")

    data := types.Service{Name: webserverName, Namespace: scaleTestNamespace, Ports: []types.ServicePort{
		{
			Name:       "http-port",
            Port:       webserverPort,
			Protocol:   "TCP",
            TargetPort: webserverPort,
		},
	}}

    exists, err := cmdExecutor.CreateService(data)
	if exists {
        printer.PrintIgnored("Service: %s already exists.", webserverName)
    } else if err != nil {
        printer.PrintErr("Error adding service %v: %s", webserverName, err)
		os.Exit(1)
	}

    printer.PrintOk("Service %s created.", webserverName)
    integration.PrettyNewLine()
}

func createScaleTestReplicationControllers() {
    printer.PrintInfo("Creating ReplicationControllers")

	vegetaRC := types.ReplicationController{Name: vegetaName, Namespace: scaleTestNamespace, Image: vegetaImage,
        Args: []types.Arg{
            {
                Key:   "-host",
                Value: webserverName,
            },
            {
                Key:   "-rate",
                Value: 1000,
            },
            {
                Key:   "-address",
                Value: ":8080",
            },
            {
                Key:   "-workers",
                Value: 10,
            },
            {
                Key:   "-duration",
                Value: "1s",
            },
        },
		ResourceRequest: types.ResourceRequest{Cpu: "100m"},
	}

    webserverRc := types.ReplicationController{Name: webserverName, Namespace: scaleTestNamespace, Image: webserverImage,
		ResourceRequest: types.ResourceRequest{Cpu: "1000m"},
	}

    err := cmdExecutor.CreateReplicationController(vegetaRC)
	if err != nil {
        printer.PrintErr("Error creating %s replication controller: %s", vegetaName, err)
		os.Exit(1)
	} else {
        printer.PrintOk("Created %s replication-controller", vegetaName)
    }

    err = cmdExecutor.CreateReplicationController(webserverRc)
	if err != nil {
        printer.PrintErr("Error creating %s replication controller: %s", webserverName, err)
		os.Exit(1)
	} else {
        printer.PrintOk("Created %s replication-controller", webserverName)
	}
}

func waitForScaleTestServicesToBeRunning() {
    printer.PrintInfo("Waiting for pods to be Running...")
	waitTime := time.Second
	done := false
	for !done {
		tmpl := "\"{..status.phase}\""
		args := []string{"--namespace=" + scaleTestNamespace, "get", "pods", "-o", "jsonpath=" + tmpl}
        sshOut, err := cmdExecutor.RunKubectlCommand(args)

		if err != nil {
            printer.PrintWarn("Error running kubectl command '%v': %s", args, err)
		}

		lines := strings.Split(sshOut.Stdout, " ")
		if len(lines) < 2 {
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

func displayScaleTestPods() {
    result, err := cmdExecutor.GetPods(scaleTestNamespace, true)
	if err != nil {
        printer.PrintWarn("Error running kubectl command '%v'", err)
	} else {
        printer.Print("Pods are running\n%s", result)
    }

    integration.PrettyNewLine()
}

func runScaleTest() {
    fetchResults()
    //cmdExecutor.ScaleReplicationController(scaleTestNamespace, webserverName, 10)
    //TODO scale scenario: run  webserver - vegeta
    // 1-1 (idle)
    // 10-1 (under-load)
    // 10-10 (equal-load)
    // 10-100 (overload-load)
    // 100-1000 (one million requests per second)
    // scale/wait, query QPS, iterate
}

func fetchResults() {
	var ips []string
	var err error
	attempts := 0

	for {
		ips, err = getLoadbotPodIPs()
		if err != nil {
            printer.PrintDebug("Could not get loadbot ips: %s", err)
			attempts += 1
			if attempts < 3 {
				time.Sleep(2 * time.Second)
				continue
			} else {
                printer.PrintErr("Failed to get loadbot ips after 3 attempts: %v", err)
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
            cmd := "curl --silent http://" + ip + ":8080/"
            resp, err := cmdExecutor.PerformCmd(cmd)
			if err != nil {
                printer.PrintWarn("Error calling %s on node %s: %s", cmd, util.GetNodeAddress(node), err)
				return
			}

			var metrics loadbotMetrics
            if err := json.Unmarshal([]byte(resp.Stdout), &metrics); err != nil {
                printer.PrintWarn("Error decoding response of %s on node %s: %v\n", cmd, util.GetNodeAddress(node), err)
				return
			}
			lock.Lock()
			defer lock.Unlock()
			parts = append(parts, metrics)
		}(ip)
	}
	wg.Wait()
	evaluateData(parts)

    printer.PrintDebug("Updated loadbots results.\n")
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
    sshOut, err := cmdExecutor.RunKubectlCommand(args)

	if err != nil {
		return []string{}, err
	}

	return strings.Split(sshOut.Stdout, " "), nil
}

func removeScaleTest() {
    name := "svc/" + webserverName
    err := cmdExecutor.RemoveResource(scaleTestNamespace, name)
	if err != nil {
        printer.PrintWarn("Error deleting service '%v'", name, err)
    }

    err = cmdExecutor.RemoveResource(scaleTestNamespace, vegetaName)
	if err != nil {
        printer.PrintWarn("Error deleting replication-controller '%v'", vegetaName, err)
    }

    err = cmdExecutor.RemoveResource(scaleTestNamespace, webserverName)
	if err != nil {
        printer.PrintWarn("Error deleting replication-controller '%v'", webserverName, err)
	}
}
