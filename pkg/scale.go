package pkg

import (
    "encoding/json"

    "github.com/mrahbar/kubernetes-inspector/ssh"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/mrahbar/kubernetes-inspector/util"
    "os"
    "path"
    "strings"
    "time"
    "fmt"
    "io/ioutil"
    "path/filepath"
)

const (
    scaleTestNamespace = "scaletest"

    loadbotsImage  = "endianogino/vegeta-server:1.0"
    webserverImage = "endianogino/simple-webserver:1.0"

    loadbotsName  = "loadbots"
    webserverName = "webserver"

    webserverPort = 80
    loadbotsPort  = 8080

    MaxScaleReplicas = 100

    fetch_metrics_script = `
echo '[' > response.json
for i in {1..10} ; do
    for var in "$@"; do
        output=$(curl --noproxy '*' --silent http://$var)
        echo "${output}" >> response.json
        echo "," >> response.json
    done;
    sleep 1;
done;
sed -i '$d' response.json
echo "]" >> response.json
cat response.json
rm response.json
exit 0
`
)

var scenarios = []replicas{
    {
        title:     "Idle",
        loadbots:  1,
        webserver: 1,
    },
    {
        title:     "Under load",
        loadbots:  1,
        webserver: 10,
    },
    {
        title:     "Equal load",
        loadbots:  10,
        webserver: 10,
    },
    {
        title:     "Over load",
        loadbots:  100,
        webserver: 10,
    },
    {
        title:     "High load",
        loadbots:  100,
        webserver: 100,
    },
}

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

type replicas struct {
    title     string
    loadbots  int
    webserver int
}

type resultEntry struct {
    title  string
    result string
}

var remoteScriptFile string
var summary []resultEntry
var scaleTestOpts *types.ScaleTestOpts

func ScaleTest(cmdParams *types.CommandContext) {
    initParams(cmdParams)
    scaleTestOpts = cmdParams.Opts.(*types.ScaleTestOpts)
    group := util.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)

    if scaleTestOpts.MaxReplicas <= 1 {
        printer.PrintCritical("Max replicas must be greater than 1 was %d", scaleTestOpts.MaxReplicas)
    }

    if group.Nodes == nil || len(group.Nodes) == 0 {
        printer.PrintCritical("No host configured for group [%s]", types.MASTER_GROUPNAME)
    }

    sshOpts = config.Ssh
    node = ssh.GetFirstAccessibleNode(config.Ssh.LocalOn, cmdExecutor, group.Nodes)

    if !util.IsNodeAddressValid(node) {
        printer.PrintCritical("No master available")
    }

    if scaleTestOpts.OutputDir == "" {
        exPath, err := util.GetExecutablePath()
        if err != nil {
            printer.PrintCritical("Could not get current executable path: %s", err)
        }
        scaleTestOpts.OutputDir = path.Join(exPath, "scaletest-results")
    }

    err := os.MkdirAll(scaleTestOpts.OutputDir, os.ModePerm)
    if err != nil {
        printer.PrintCritical("Failed to open output file for path %s Error: %v", scaleTestOpts.OutputDir, err)
    }

    printer.Print("Running kubectl commands on node %s", util.ToNodeLabel(node))
    cmdExecutor.SetNode(node)

    checkingScaleTestPreconditions()
    deployScript()
    createScaleTestNamespace()
    createScaleTestServices()
    createScaleTestReplicationControllers()

    runScaleTest()
    showSummary()

    if scaleTestOpts.Cleanup {
        printer.PrintInfo("Cleaning up...")
        removeScaleTest()
    } else {
        cmdExecutor.ScaleReplicationController(scaleTestNamespace, loadbotsName, 1)
        cmdExecutor.ScaleReplicationController(scaleTestNamespace, webserverName, 1)
    }

    printer.PrintOk("DONE")
}

func checkingScaleTestPreconditions() {
    count, err := cmdExecutor.GetNumberOfReadyNodes()

    if err != nil {
        printer.PrintCritical("Error checking node count: %s", err)
    } else if count < 1 {
        printer.PrintErr("Insufficient number of nodes for scale test (need minimum of 1 node)")
        os.Exit(2)
    }
}

func createScaleTestNamespace() {
    printer.PrintInfo("Creating namespace")
    err := cmdExecutor.CreateNamespace(scaleTestNamespace)

    if err != nil {
        printer.PrintCritical("Error creating test namespace: %s", err)
    } else {
        printer.PrintOk("Namespace %s created", scaleTestNamespace)
    }
    printer.PrintNewLine()
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
        printer.PrintCritical("Error adding service %v: %s", webserverName, err)
    }

    printer.PrintOk("Service %s created.", webserverName)
    printer.PrintNewLine()
}

func createScaleTestReplicationControllers() {
    printer.PrintInfo("Creating ReplicationControllers")

    loadbotsRC := types.ReplicationController{Name: loadbotsName, Namespace: scaleTestNamespace, Image: loadbotsImage,
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
        Ports: []types.PodPort{
            {
                Name:     "http-port",
                Port:     loadbotsPort,
                Protocol: "TCP",
            },
        },
    }

    webserverRc := types.ReplicationController{Name: webserverName, Namespace: scaleTestNamespace, Image: webserverImage,
        ResourceRequest: types.ResourceRequest{Cpu: "100m"},
        Args: []types.Arg{
            {
                Key:   "-port",
                Value: webserverPort,
            },
        },
        Ports: []types.PodPort{
            {
                Name:     "http-port",
                Port:     webserverPort,
                Protocol: "TCP",
            },
        },
    }

    if err := cmdExecutor.CreateReplicationController(webserverRc); err != nil {
        printer.PrintCritical("Error creating %s replication controller: %s", webserverName, err)
    } else {
        printer.PrintOk("Created %s replication-controller", webserverName)
    }

    if err := cmdExecutor.CreateReplicationController(loadbotsRC); err != nil {
        printer.PrintCritical("Error creating %s replication controller: %s", loadbotsName, err)
    } else {
        printer.PrintOk("Created %s replication-controller", loadbotsName)
    }

    printer.PrintNewLine()
}

func deployScript() {
    tmpFile, err := ioutil.TempFile("", "kubespector-")
    if err != nil {
        printer.PrintCritical("Could not create temporary file: %s", err)
    }

    if err := ioutil.WriteFile(tmpFile.Name(), []byte(fetch_metrics_script), os.ModeAppend); err != nil {
        printer.PrintCritical("Could not write to temporary file: %s", err)
    }

    remoteScriptFile = path.Join("/tmp", filepath.Base(tmpFile.Name()))
    if err := cmdExecutor.UploadFile(remoteScriptFile, tmpFile.Name()); err != nil {
        printer.PrintCritical("Could not deploy fetch_metrics_script: %s", err)
    }

    if _, err := cmdExecutor.PerformCmd(fmt.Sprintf("chmod +x %s", remoteScriptFile)); err != nil {
        printer.PrintCritical("Could not set remote script file %s to executable: %s", remoteScriptFile, err)
    }
}

func waitForScaleTestServicesToBeRunning(targets int) {
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
        if len(lines) < targets {
            printer.Print("Pods status output too short. Waiting %v then checking again.", waitTime)
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
            printer.Print("Pods not running. Waiting %v then checking again.", waitTime)
            time.Sleep(waitTime)
            waitTime *= 2
        } else {
            done = true
        }
    }
}

func runScaleTest() {
    var currentLoadbots, currentWebservers int

    for _, s := range scenarios {
        var queryPerSecond, success float64
        var latencyMean, latency99th time.Duration

        loadbotReplicas := s.loadbots * scaleTestOpts.MaxReplicas /100
        webserverReplicas := s.webserver * scaleTestOpts.MaxReplicas /100
        if s.loadbots != 1 {
            time.Sleep(1 * time.Second)
            if currentLoadbots != loadbotReplicas {
                cmdExecutor.ScaleReplicationController(scaleTestNamespace, loadbotsName, loadbotReplicas)
                currentLoadbots = loadbotReplicas
            }
        } else {
            currentLoadbots = 1
            loadbotReplicas = 1
        }

        if s.webserver != 1 {
            time.Sleep(1 * time.Second)
            if currentWebservers != webserverReplicas {
                cmdExecutor.ScaleReplicationController(scaleTestNamespace, webserverName, webserverReplicas)
                currentWebservers = webserverReplicas
            }
        } else {
            currentWebservers = 1
            webserverReplicas = 1
        }

        printer.PrintInfo("Load scenario '%s': %d Loadbots - %d Webservers", s.title, loadbotReplicas, webserverReplicas)

        waitForScaleTestServicesToBeRunning(currentWebservers + currentLoadbots)
        time.Sleep(3 * time.Second)
        res, err := cmdExecutor.GetPods(scaleTestNamespace, true)
        if err != nil {
            printer.PrintWarn("Error running kubectl command '%v'", err)
        } else {
            printer.Print("Pods are running. Fetching metrics")
            printer.PrintDebug("%s\n", res.Stdout)
        }

        attempts := 0
        for {
            parts, err := fetchResults()
            if err != nil || len(parts) == 0 {
                printer.PrintDebug("Could not run load scenario '%s': %s", s.title, err)
                attempts += 1
                if attempts < 3 {
                    time.Sleep(1 * time.Second)
                    continue
                } else {
                    printer.PrintCritical("Failed to run load scenario '%s' after 3 attempts: %s", s.title, err)
                }
            } else {
                queryPerSecond, success, latencyMean, latency99th = evaluateData(parts)
                break
            }
        }

        result := fmt.Sprintf("QPS: %-8.0f Success: %-8.2f%s Latency: %s (mean) %s (99th)",
            queryPerSecond, success, "%%", latencyMean, latency99th)
        summary = append(summary, resultEntry{
            title:  s.title,
            result: result,
        })

        printer.PrintOk("Summary of load scenario '%s':\n%s", s.title, result)
        printer.PrintNewLine()
    }
}

func fetchResults() ([]loadbotMetrics, error) {
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
                printer.PrintWarn("Failed to get loadbot ips after 3 attempts: %v", err)
                return []loadbotMetrics{}, err
            }
        } else {
            break
        }
    }

    ipsWithPort := []string{}
    for _, v := range ips {
        ipsWithPort = append(ipsWithPort, fmt.Sprintf("%s:%d", v, loadbotsPort))
    }
    cmd := fmt.Sprintf("%s %s", remoteScriptFile, strings.Join(ipsWithPort, " "))
    printer.PrintDebug("Executing script file: %s", cmd)
    var parts []loadbotMetrics
    response := "[]"
    attempts = 0

    for {
        resp, e := cmdExecutor.PerformCmd(cmd)
        if e != nil {
            printer.PrintDebug("Could execute %s: %s", remoteScriptFile, e)
            attempts += 1
            if attempts < 3 {
                time.Sleep(1 * time.Second)
                continue
            } else {
                printer.PrintWarn("Failed to get metrics after 3 attempts: %v", e)
                return []loadbotMetrics{}, err
            }
        } else {
            response = resp.Stdout
            break
        }
    }

    printer.PrintDebug("Loadbot response: %s", response)
    if err := json.Unmarshal([]byte(response), &parts); err != nil {
        printer.PrintWarn("Error decoding response of %s: %v\n", cmd, err)
        return []loadbotMetrics{}, err
    }

    return parts, nil
}

func evaluateData(metrics []loadbotMetrics) (queryPerSecond float64, success float64,
    latencyMean time.Duration, latency99th time.Duration) {

    var latencyMeans time.Duration
    var latency99ths time.Duration

    for _, v := range metrics {
        if v.Rate > 0 {
            queryPerSecond += v.Rate
        }

        success += v.Success * 100
        latencyMeans += v.Latencies.Mean
        latency99ths += v.Latencies.P99
    }

    success /= float64(len(metrics))
    latencyMean = time.Duration(latencyMeans.Nanoseconds() / int64(len(metrics)))
    latency99th = time.Duration(latency99ths.Nanoseconds() / int64(len(metrics)))

    printer.PrintDebug("%s: QPS: %.0f Success: %.2f%s - Latency mean: %s 99th: %s",
        time.Now().Format("2006-01-02T15:04:05"), queryPerSecond, success, "%%", latencyMean, latency99th)

    return queryPerSecond, success, latencyMean, latency99th
}

func showSummary() {
    printer.PrintOk("Summary of load scenarios:")
    for k, s := range summary {
        printer.Print("%d. %-10s: %s", k, s.title, s.result)
    }
    printer.PrintNewLine()
}

func getLoadbotPodIPs() ([]string, error) {
    tmpl := "\"{..status.podIP}\""
    args := []string{"--namespace=" + scaleTestNamespace, "get", "pods", "-l", "app=" + loadbotsName, "-o", "jsonpath=" + tmpl}
    sshOut, err := cmdExecutor.RunKubectlCommand(args)

    if err != nil {
        return []string{}, err
    }

    return strings.Split(sshOut.Stdout, " "), nil
}

func removeScaleTest() {
    if err := cmdExecutor.RemoveResource(scaleTestNamespace, "svc/"+webserverName); err != nil {
        printer.PrintWarn("Error deleting service '%v': %s", webserverName, err)
    }
    if err := cmdExecutor.RemoveResource(scaleTestNamespace, "svc/"+webserverName); err != nil {
        printer.PrintWarn("Error deleting service '%v': %s", webserverName, err)
    }

    if err := cmdExecutor.RemoveResource(scaleTestNamespace, "rc/"+loadbotsName); err != nil {
        printer.PrintWarn("Error deleting replication-controller '%v': %s", loadbotsName, err)
    }

    if err := cmdExecutor.RemoveResource(scaleTestNamespace, "rc/"+webserverName); err != nil {
        printer.PrintWarn("Error deleting replication-controller '%v': %s", webserverName, err)
    }

    if err := cmdExecutor.RemoveResource("", scaleTestNamespace); err != nil {
        printer.PrintWarn("Error deleting namespace '%v': %s", scaleTestNamespace, err)
    }

    if err := cmdExecutor.DeleteRemoteFile(remoteScriptFile); err != nil {
        printer.PrintWarn("Error deleting remote script file '%v': %s", remoteScriptFile, err)
    }
}
