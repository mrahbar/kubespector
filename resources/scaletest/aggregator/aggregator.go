package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "net/http"
    "sync"
    "time"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    apiv1 "k8s.io/client-go/pkg/api/v1"

    "github.com/tsenart/vegeta/lib"
    "strings"
    "log"
    "os"
)

type replicas struct {
    title     string
    loadbots  int32
    webserver int32
}

type resultEntry struct {
    title  string
    result string
}

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

var summary []resultEntry
var clientset *kubernetes.Clientset

const (
    summaryDataMarker    = "GENERATING SUMMARY OUTPUT"
    summaryEndDataMarker = "END SUMMARY DATA"

    loadbotsName  = "loadbots"
    webserverName = "webserver"

    maxScaleReplicas = 100
    iterations = 10
)

var (
    scaleTestNamespace = "default"

    selector     = flag.String("selector", "app", "The label key as selector for pods")
    loadbotsPort = flag.Int("loadbots-port", 8080, "Target port of selected pods")
    maxReplicas  = int32(flag.Int("max-replicas", maxScaleReplicas, "Maximum replication count per service. Total replicas will be twice as much."))
    useIP        = flag.Bool("use-ip", true, "Use IP for aggregation")
    sleep        = flag.Duration("sleep", 1*time.Second, "The sleep period between aggregations")
)

func main() {
    flag.Parse()
    createKubernetesClient()
    setNamespace()

    log.Println("Running scale test")
    runScaleTest()
    showSummary()
}

func createKubernetesClient() {
    config, err := rest.InClusterConfig()
    if err != nil {
        panic(err.Error())
    }

    clientset, err = kubernetes.NewForConfig(config)
    if err != nil {
        panic(err.Error())
    }

    v, err := clientset.Discovery().ServerVersion()
    if err != nil {
        panic(err.Error())
    }

    log.Printf("Running in Kubernetes Cluster version v%v.%v (%v) - git (%v) commit %v - platform %v",
        v.Major, v.Minor, v.GitVersion, v.GitTreeState, v.GitCommit, v.Platform)
}

func setNamespace()  {
    if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
        scaleTestNamespace = ns
    }

    // Fall back to the namespace associated with the service account token, if available
    if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
        if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
            scaleTestNamespace = ns
        }
    }

    scaleTestNamespace = "default"
}

func runScaleTest() {
    var currentLoadbots, currentWebservers int32

    for _, s := range scenarios {
        var queryPerSecond, success float64
        var latencyMean, latency99th time.Duration
        var latencyMeans, latency99ths time.Duration

        loadbotReplicas := s.loadbots * maxReplicas / 100
        webserverReplicas := s.webserver * maxReplicas / 100
        if s.loadbots != 1 {
            time.Sleep(1 * time.Second)
            if currentLoadbots != loadbotReplicas {
                scaleReplicationController(scaleTestNamespace, loadbotsName, loadbotReplicas)
                currentLoadbots = loadbotReplicas
            }
        } else {
            currentLoadbots = 1
            loadbotReplicas = 1
        }

        if s.webserver != 1 {
            time.Sleep(1 * time.Second)
            if currentWebservers != webserverReplicas {
                scaleReplicationController(scaleTestNamespace, webserverName, webserverReplicas)
                currentWebservers = webserverReplicas
            }
        } else {
            currentWebservers = 1
            webserverReplicas = 1
        }

        log.Printf("Load scenario '%s': %d Loadbots - %d Webservers", s.title, loadbotReplicas, webserverReplicas)

        waitForScaleTestServicesToBeRunning(currentWebservers + currentLoadbots)
        time.Sleep(5 * time.Second)

        parts := []vegeta.Metrics{}
        loadbots, err := getPods(loadbotsName)
        if err != nil {
            log.Printf("Error getting loadbot pods: %s", err)
        }

        for i := 1; i <= iterations; i++ {
            attempts := 0
            for {
                start := time.Now()
                partsIteration, err := fetchResults(loadbots)
                if err != nil || len(parts) == 0 {
                    attempts += 1
                    if attempts < 3 {
                        time.Sleep(1 * time.Second)
                        continue
                    } else {
                        log.Printf("Failed to run load scenario '%s' after 3 attempts: %s", s.title, err)
                    }
                } else {
                    parts = append(parts, partsIteration...)
                    latency := time.Since(start)
                    if latency < *sleep {
                        time.Sleep(*sleep - latency)
                    }
                    break
                }
            }
        }

        qps, scs, lm, lp99 := evaluateData(parts)
        queryPerSecond += qps
        success += scs
        latencyMeans += lm
        latency99ths += lp99

        success /= float64(iterations)
        latencyMean = time.Duration(latencyMeans.Nanoseconds() / int64(iterations))
        latency99th = time.Duration(latency99ths.Nanoseconds() / int64(iterations))

        result := fmt.Sprintf("QPS: %-8.0f Success: %-8.2f%s Latency: %s (mean) %s (99th)",
            queryPerSecond, success, "%%", latencyMean, latency99th)

        summary = append(summary, resultEntry{
            title:  s.title,
            result: result,
        })

        fmt.Printf("Summary of load scenario '%s':\n%s\n", s.title, result)
        fmt.Println("")
    }
}

func fetchResults(loadbots []*apiv1.Pod) ([]vegeta.Metrics, error) {
    parts := []vegeta.Metrics{}
    lock := sync.Mutex{}
    wg := sync.WaitGroup{}
    wg.Add(len(loadbots))
    for ix := range loadbots {
        go func(ix int) {
            defer wg.Done()
            pod := loadbots[ix]
            var data []byte
            if *useIP {
                url := "http://" + pod.Status.PodIP + ":"+string(loadbotsPort)+"/"
                resp, err := http.Get(url)
                if err != nil {
                    fmt.Printf("Error getting: %v\n", err)
                    return
                }
                defer resp.Body.Close()
                if data, err = ioutil.ReadAll(resp.Body); err != nil {
                    fmt.Printf("Error reading: %v\n", err)
                    return
                }
            } else {
                var err error

                data, err = clientset.Discovery().RESTClient().Get().AbsPath("/api/v1/proxy/namespaces/default/pods/" + pod.Name + ":"+string(loadbotsPort)+"/").DoRaw()
                if err != nil {
                    fmt.Printf("Error proxying to pod: %v\n", err)
                    return
                }
            }
            var metrics vegeta.Metrics
            if err := json.Unmarshal(data, &metrics); err != nil {
                fmt.Printf("Error decoding: %v\n", err)
                return
            }
            lock.Lock()
            defer lock.Unlock()
            parts = append(parts, metrics)
        }(ix)
    }
    wg.Wait()
    return parts, nil
}

func evaluateData(metrics []vegeta.Metrics) (queryPerSecond float64, success float64, latencyMean time.Duration, latency99th time.Duration) {
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

    fmt.Printf("%s: QPS: %.0f Success: %.2f%s - Latency mean: %s 99th: %s\n",
        time.Now().Format("2006-01-02T15:04:05"), queryPerSecond, success, "%%", latencyMean, latency99th)

    return queryPerSecond, success, latencyMean, latency99th
}

func showSummary() {
    fmt.Printf("Summary of load scenarios: %s\n", summaryDataMarker)
    for k, s := range summary {
        log.Printf("%d. %-10s: %s", k, s.title, s.result)
    }
    fmt.Printf("%s\n", summaryEndDataMarker)
}

func getPods(appName string) ([]*apiv1.Pod, error) {
    loadbots := []*apiv1.Pod{}

    pods, err := clientset.CoreV1().Pods(scaleTestNamespace).List(metav1.ListOptions{
        LabelSelector: fmt.Sprintf("%s=%s", *selector, appName),
    })
    if err != nil {
        return loadbots, err
    }

    for ix := range pods.Items {
        pod := &pods.Items[ix]
        if pod.Status.PodIP == "" {
            continue
        }
        loadbots = append(loadbots, pod)
    }

    return loadbots, nil
}

func waitForScaleTestServicesToBeRunning(target int32) {
    waitTime := time.Second
    done := false
    for !done {
        loadbotPods, err := clientset.CoreV1().Pods(scaleTestNamespace).List(metav1.ListOptions{
            LabelSelector: fmt.Sprintf("%s=%s", *selector, loadbotsName),
        })
        if err != nil {
            log.Printf("Error getting list of loadbots: %s", err)
        }
        webserverPods, err := clientset.CoreV1().Pods(scaleTestNamespace).List(metav1.ListOptions{
            LabelSelector: fmt.Sprintf("%s=%s", *selector, webserverName),
        })
        if err != nil {
            log.Printf("Error getting list of webservers: %s", err)
        }

        lines := int32(len(loadbotPods.Items) + len(webserverPods.Items))
        if lines < target {
            time.Sleep(waitTime)
            waitTime *= 2
            continue
        }

        allRunning := true
        for _, p := range append(loadbotPods.Items, webserverPods.Items...) {
            if p.Status.Phase != apiv1.PodRunning {
                allRunning = false
                break
            }
        }
        if !allRunning {
            time.Sleep(waitTime)
            waitTime *= 2
        } else {
            done = true
        }
    }
}

func scaleReplicationController(namespace string, name string, replicas int32) error {
    rc, err := clientset.ReplicationControllers(namespace).Get(name, metav1.GetOptions{})
    if err != nil {
        return err
    }
    rc.Spec.Replicas = &replicas
    clientset.ReplicationControllers(namespace).Update(rc)
    return nil
}
