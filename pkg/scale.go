package pkg

import (

    "github.com/mrahbar/kubernetes-inspector/ssh"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/mrahbar/kubernetes-inspector/util"
    "os"
    "path"
    "strings"
    "time"
)

const (
    scaleTestNamespace = "scaletest"

    aggregatorImage  = "endianogino/vegeta-aggregator:1.0"
    loadbotsImage  = "endianogino/vegeta-server:1.0"
    webserverImage = "endianogino/simple-webserver:1.0"

    loadbotsName  = "loadbots"
    webserverName = "webserver"
    aggregatorName = "aggregator"

    webserverPort = 80
    loadbotsPort  = 8080
)

var scaleTestOpts *types.ScaleTestOpts

func ScaleTest(cmdParams *types.CommandContext) {
    initParams(cmdParams)
    scaleTestOpts = cmdParams.Opts.(*types.ScaleTestOpts)
    group := util.FindGroupByName(config.ClusterGroups, types.MASTER_GROUPNAME)

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
    createScaleTestNamespace()
    createScaleTestServices()
    createScaleTestReplicationControllers()
    waitForScaleTestServicesToBeRunning()
    displayScaleTestPods()

    if scaleTestOpts.Cleanup {
        printer.PrintInfo("Cleaning up...")
        removeScaleTest()
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
        ResourceRequest: types.ResourceRequest{Cpu: "1000m"},
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

    aggregatorRc := types.ReplicationController{Name: aggregatorName, Namespace: scaleTestNamespace, Image: aggregatorImage,
        ResourceRequest: types.ResourceRequest{Cpu: "500m"},
        Args: []types.Arg{
            {
                Key:   "-sleep",
                Value: "1s",
            },
            {
                Key:   "-use-ip",
                Value: true,
            },
            {
                Key:   "-selector",
                Value: "app=loadbots",
            },
            {
                Key:   "-loadbots-port",
                Value: loadbotsPort,
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

    if err := cmdExecutor.CreateReplicationController(aggregatorRc); err != nil {
        printer.PrintCritical("Error creating %s replication controller: %s", aggregatorName, err)
    } else {
        printer.PrintOk("Created %s replication-controller", aggregatorName)
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
        if len(lines) < 3 {
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
    printer.PrintNewLine()
}

func displayScaleTestPods() {
    result, err := cmdExecutor.GetPods(scaleTestNamespace, true)
    if err != nil {
        printer.PrintWarn("Error running kubectl command '%v'", err)
    } else {
        printer.Print("Pods are running\n%s", result.Stdout)
    }

    printer.PrintNewLine()
}

func removeScaleTest() {
    if err := cmdExecutor.RemoveResource(scaleTestNamespace, "svc/"+webserverName); err != nil {
        printer.PrintWarn("Error deleting service '%v': %s", webserverName, err)
    }

    if err := cmdExecutor.RemoveResource(scaleTestNamespace, "rc/"+loadbotsName); err != nil {
        printer.PrintWarn("Error deleting replication-controller '%v': %s", loadbotsName, err)
    }

    if err := cmdExecutor.RemoveResource(scaleTestNamespace, "rc/"+webserverName); err != nil {
        printer.PrintWarn("Error deleting replication-controller '%v': %s", webserverName, err)
    }

    if err := cmdExecutor.RemoveResource(scaleTestNamespace, "rc/"+aggregatorName); err != nil {
        printer.PrintWarn("Error deleting replication-controller '%v': %s", aggregatorName, err)
    }
}
