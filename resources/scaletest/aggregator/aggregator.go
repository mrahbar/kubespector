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
    apiv1 "k8s.io/client-go/pkg/api/v1"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"

    "github.com/tsenart/vegeta/lib"
    "log"
    "os"
    "path/filepath"
    "strings"
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
var kubeconfig *string
var scaleTestNamespace string
var clientset *kubernetes.Clientset

const (
    summaryDataMarker    = "GENERATING SUMMARY OUTPUT"
    summaryEndDataMarker = "END SUMMARY DATA"

    loadbotsName  = "loadbots"
    webserverName = "webserver"

    maxScaleReplicas = 100
    iterations       = 10
)

var (
    inCluster    = flag.Bool("incluster", true, "Running aggregator inside Kubernetes")
    selector     = flag.String("selector", "app", "The label key as selector for pods")
    loadbotsPort = flag.Int("loadbots-port", 8080, "Target port of selected pods")
    maxReplicas  = flag.Int("max-replicas", maxScaleReplicas, "Maximum replication count per service. Total replicas will be twice as much.")
    sleep        = flag.Duration("sleep", 1*time.Second, "The sleep period between aggregations")
    debug        = flag.Bool("debug", false, "Increase debugging output")
)

func main() {
    if home := homeDir(); home != "" {
        kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file when in-cluster false")
    } else {
        kubeconfig = flag.String("kubeconfig", "", "(optional) absolute path to the kubeconfig file when in-cluster false")
    }

    flag.Parse()
    createKubernetesClient()
    setNamespace()

    log.Println("Running preflight checks")
    preflightChecks()
    log.Println("Finished preflight checks")

    log.Printf("Running scale test with max replicas %d", *maxReplicas)
    runScaleTest()
    showSummary()
    scaleReplicationController(scaleTestNamespace, loadbotsName, 0)
    scaleReplicationController(scaleTestNamespace, webserverName, 0)
}

func createKubernetesClient() {
    var clientsetError error
    if *inCluster {
        config, err := rest.InClusterConfig()
        if err == nil {
            panic(err.Error())
        }
        clientset, clientsetError = kubernetes.NewForConfig(config)
    } else {
        config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
        if err != nil {
            panic(err.Error())
        }

        clientset, clientsetError = kubernetes.NewForConfig(config)
    }

    if clientsetError != nil {
        panic(clientsetError.Error())
    }

    v, err := clientset.Discovery().ServerVersion()
    if err != nil {
        panic(err.Error())
    }

    inClusterConf := ""
    if *inCluster {
        inClusterConf = "in"
    } else {
        inClusterConf = "out of"
    }

    log.Printf("Running %s Kubernetes Cluster - version v%v.%v (%v) - platform %v",
        inClusterConf, v.Major, v.Minor, v.GitVersion, v.Platform)
}

func setNamespace() {
    if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
        scaleTestNamespace = ns
    } else if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
        if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
            scaleTestNamespace = ns
        }
    }

    if scaleTestNamespace == "" {
        scaleTestNamespace = "default"
    }

    log.Printf("Running aggregator in namespace %s", scaleTestNamespace)
}

func preflightChecks() {
    log.Printf("Waiting for initial loadbot and webserver pods to be Running...")
    waitForScaleTestServicesToBeRunning(1,1)
}

func runScaleTest() {
    var currentLoadbots, currentWebservers int32

    for _, s := range scenarios {
        loadbotReplicas := s.loadbots * int32(*maxReplicas) / 100
        webserverReplicas := s.webserver * int32(*maxReplicas) / 100
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
        waitForScaleTestServicesToBeRunning(currentLoadbots, currentWebservers)
        time.Sleep(5 * time.Second)

        parts := []vegeta.Metrics{}
        if *debug {
            log.Printf("[D] Getting %s pods", loadbotsName)
        }
        loadbots, err := getPods(loadbotsName)
        if err != nil {
            log.Printf("Error getting loadbot pods: %s", err)
        } else if *debug {
            log.Printf("[D] Got %d pods", len(loadbots))
        }

        for i := 1; i <= iterations; i++ {
            attempts := 0
            for {
                start := time.Now()
                partsIteration, err := fetchResults(loadbots)
                if err != nil || len(partsIteration) == 0 {
                    attempts += 1
                    if attempts < 3 {
                        if *debug {
                            log.Println ("[D] Failed to fetch results. Trying again")
                        }
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

        if *debug {
            log.Printf("[D] Fetched results:\n %s", parts)
        }

        queryPerSecond, success, latencyMean, latency99th := evaluateData(parts)

        result := fmt.Sprintf("QPS: %-8.0f Success: %-8.2f%% Latency: %s (mean) %s (99th)",
            queryPerSecond, success, latencyMean, latency99th)

        summary = append(summary, resultEntry{
            title:  s.title,
            result: result,
        })

        log.Printf("Summary of load scenario '%s': %s", s.title, result)
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
            if *inCluster {
                url := fmt.Sprintf("http://%s:%d/", pod.Status.PodIP, *loadbotsPort)
                resp, err := http.Get(url)
                if err != nil {
                    log.Printf("Error getting %s: %v", url, err)
                    return
                }
                defer resp.Body.Close()
                if data, err = ioutil.ReadAll(resp.Body); err != nil {
                    log.Printf("Error reading response of %s: %v", url, err)
                    return
                }
            } else {
                var err error

                url := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s:%d/proxy", scaleTestNamespace, pod.Name, *loadbotsPort)
                data, err = clientset.Discovery().RESTClient().Get().AbsPath(url).DoRaw()
                if err != nil {
                    log.Printf("Error proxying to pod %s: %v", url, err)
                    return
                }
            }
            var metrics vegeta.Metrics
            if err := json.Unmarshal(data, &metrics); err != nil {
                log.Printf("Error decoding: %v\n", err)
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

    return queryPerSecond, success, latencyMean, latency99th
}

func showSummary() {
    log.Println("Summary of load scenarios:")
    log.Println(summaryDataMarker)
    for k, s := range summary {
        log.Printf("%d. %-10s: %s", k, s.title, s.result)
    }
    log.Printf("%s\n", summaryEndDataMarker)
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
        if pod.Status.PodIP == "" || pod.Status.Phase != apiv1.PodRunning {
            continue
        }
        loadbots = append(loadbots, pod)
    }

    return loadbots, nil
}

func waitForScaleTestServicesToBeRunning(targetLoadbots int32, targetWebserver int32) {
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
        if lines < targetLoadbots + targetWebserver {
            log.Printf("Pods status output too short. Waiting %v then checking again.", waitTime)
            time.Sleep(waitTime)
            waitTime *= 2
            continue
        }

        loadbotsRunning := false
        webserverRunning := false
        totalLoadbotsRunning := 0
        totalWebserverRunning := 0
        for _, p := range loadbotPods.Items {
            if p.Status.Phase == apiv1.PodRunning {
                totalLoadbotsRunning++
                if int32(totalLoadbotsRunning) >= targetLoadbots {
                    loadbotsRunning = true
                    break
                }
            }
        }
        for _, p := range  webserverPods.Items {
            if p.Status.Phase == apiv1.PodRunning {
                totalWebserverRunning++
                if int32(totalWebserverRunning) >= targetWebserver {
                    webserverRunning = true
                    break
                }
            }
        }
        if !loadbotsRunning || !webserverRunning {
            log.Printf("Pods are not running. Waiting %v then checking again.", waitTime)
            time.Sleep(waitTime)
            waitTime *= 2
        } else {
            done = true
        }
    }
}

func scaleReplicationController(namespace string, name string, replicas int32) error {
    log.Printf("Scaling %s to %d replicas", name, replicas)
    rc, err := clientset.CoreV1().ReplicationControllers(namespace).Get(name, metav1.GetOptions{})
    if err != nil {
        log.Printf("Error scaling %s to %d replicas: %s", name, replicas, err)
        return err
    }

    rc.Spec.Replicas = &replicas
    _, err = clientset.CoreV1().ReplicationControllers(namespace).Update(rc)
    if err != nil {
        log.Printf("Error scaling %s to %d replicas: %s", name, replicas, err)
        return err
    }

    return nil
}

func homeDir() string {
    if h := os.Getenv("HOME"); h != "" {
        return h
    }
    return os.Getenv("USERPROFILE") // windows
}
