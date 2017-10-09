package pkg

import (
    "bytes"
    "fmt"
    "github.com/mrahbar/kubernetes-inspector/ssh"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/mrahbar/kubernetes-inspector/util"
    "regexp"
    "sort"
    "strconv"
    "strings"
    "text/template"
    "time"
)

const (
    leftTemplateDelim  = "{{"
    rightTemplateDelim = "}}"
)

var clusterStatusChecks = []string{types.SERVICES_CHECKNAME, types.CONTAINERS_CHECKNAME, types.CERTIFICATES_CHECKNAME, types.DISKUSAGE_CHECKNAME, types.KUBERNETES_CHECKNAME}
var clusterStatusOpts = &types.ClusterStatusOpts{}

func ClusterStatus(cmdParams *types.CommandContext) {
    initParams(cmdParams)
    clusterStatusOpts = cmdParams.Opts.(*types.ClusterStatusOpts)

    var groups = []string{}
    if strings.EqualFold(clusterStatusOpts.Groups, types.ALL_GROUPNAME) {
        for _, group := range config.ClusterGroups {
            groups = append(groups, group.Name)
        }
    } else if clusterStatusOpts.Groups != "" {
        groups = strings.Split(clusterStatusOpts.Groups, ",")
    } else {
        printer.PrintCritical("No group specified")
    }

    if clusterStatusOpts.Checks != "" {
        clusterStatusChecks = strings.Split(clusterStatusOpts.Checks, ",")
    }

    printer.Print("Performing status checks %s for groups: %v",
        strings.Join(clusterStatusChecks, ","), strings.Join(groups, " "))

    totalNodes := []types.Node{}
    for _, element := range groups {
        group := util.FindGroupByName(config.ClusterGroups, element)

        for _, n := range group.Nodes {
            if util.IsNodeAddressValid(n) && !util.NodeInArray(totalNodes, n) {
                totalNodes = append(totalNodes, n)
            }
        }
    }
    sort.Slice(totalNodes, func(i, j int) bool { //TODO fix ordering
        return util.GetNodeAddress(totalNodes[i]) < util.GetNodeAddress(totalNodes[j])
    })
    printer.PrintHeader(fmt.Sprintf("Retrieving node stats"), '=')
    printer.PrintNewLine()
    for _, node := range totalNodes {
        getNodesStats(node)
    }

    for _, g := range groups {
        group := util.FindGroupByName(config.ClusterGroups, g)
        printer.PrintHeader(fmt.Sprintf("Group %s", g), '=')
        if group.Nodes != nil {
            if util.ElementInArray(clusterStatusChecks, types.SERVICES_CHECKNAME) {
                checkServiceStatus(g, group.Services, group.Nodes)
            }

            if util.ElementInArray(clusterStatusChecks, types.CONTAINERS_CHECKNAME) {
                checkContainerStatus(g, group.Containers, group.Nodes)
            }

            if util.ElementInArray(clusterStatusChecks, types.CERTIFICATES_CHECKNAME) {
                checkCertificatesExpiration(g, group.Certificates, group.Nodes)
            }

            if util.ElementInArray(clusterStatusChecks, types.DISKUSAGE_CHECKNAME) {
                checkDiskStatus(g, group.DiskUsage, group.Nodes)
            }

            if util.ElementInArray(clusterStatusChecks, types.KUBERNETES_CHECKNAME) {
                checkKubernetesStatus(g, group.Kubernetes, group.Nodes)
            }
        } else {
            printer.PrintErr("No Nodes found for group: %s", g)
        }
    }
}

func getNodesStats(node types.Node) {
    if !util.IsNodeAddressValid(node) {
        printer.PrintErr("Current node %q has no valid address", node)
        return
    }
    printer.Print("On node %s:", util.ToNodeLabel(node))
    cmdExecutor.SetNode(node)

    sshOut, err := cmdExecutor.PerformCmd("/bin/cat /proc/uptime", false)
    if err != nil {
        printer.PrintWarn("Could not get uptime for: %s", err)
    } else {
        parts := strings.Fields(sshOut.Stdout)
        if len(parts) == 2 {
            var upsecs float64
            upsecs, err = strconv.ParseFloat(parts[0], 64)
            if err != nil {
                printer.PrintWarn("Could not parse uptime: %s", err)
            } else {
                dur := time.Duration(upsecs * 1e9)
                dur = dur - (dur % time.Second)
                var days int
                for dur.Hours() > 24.0 {
                    days++
                    dur -= 24 * time.Hour
                }
                s1 := dur.String()
                uptimeFormated := ""
                if days > 0 {
                    uptimeFormated = fmt.Sprintf("%dd ", days)
                }
                for _, ch := range s1 {
                    uptimeFormated += string(ch)
                    if ch == 'h' || ch == 'm' {
                        uptimeFormated += " "
                    }
                }
                printer.Print("Uptime %s", uptimeFormated)
            }
        }
    }

    sshOut, err = cmdExecutor.PerformCmd("/bin/cat /proc/loadavg", false)
    if err != nil {
        printer.PrintWarn("Could not get load statistics: %s", err)
    } else {
        parts := strings.Fields(sshOut.Stdout)
        if len(parts) == 5 {
            loadMsg := fmt.Sprintf("Load periods 1m: %s - 5m: %s - 10m: %s\n", parts[0], parts[1], parts[2])

            if i := strings.Index(parts[3], "/"); i != -1 {
                totalProcs := "-"
                if i+1 < len(parts[3]) {
                    totalProcs = parts[3][i+1:]
                }
                loadMsg = fmt.Sprintf("%sNumber of total processes %s", loadMsg, totalProcs)

                printer.PrintOk(strings.TrimSpace(loadMsg))
            }
        }
    }
    printer.PrintNewLine()
}

func checkServiceStatus(group string, services []string, nodes []types.Node) {
    printer.PrintHeader(fmt.Sprintf("Checking service status in group [%s]", group), '-')
    if nodes == nil || len(nodes) == 0 {
        printer.PrintSkipped("No host configured for [%s]", group)
        return
    }
    if services == nil || len(services) == 0 {
        printer.PrintSkipped("No services configured for [%s]", group)
        return
    }

    for _, node := range nodes {
        if !util.IsNodeAddressValid(node) {
            printer.PrintErr("Current node %q has no valid address", node)
            break
        }

        printer.PrintNewLine()
        printer.Print("On node %s:", util.ToNodeLabel(node))
        cmdExecutor.SetNode(node)

        for _, service := range services {
            sshOut, err := cmdExecutor.PerformCmd(fmt.Sprintf("systemctl is-active %s", service), clusterStatusOpts.Sudo)

            if err != nil {
                printer.PrintErr("Error checking status of %s: %s", service, err)
            } else {
                result := sshOut.Stdout
                if result == "active" {
                    printer.PrintOk("Service %s is active", service)
                } else if result == "activating" || result == "inactive" {
                    printer.PrintWarn("Service %s is %s", service, result)
                } else if result == "failed" {
                    printer.PrintErr("Service %s is failed", service)
                } else {
                    printer.PrintUnknown("Service %s is unknown state: %s", service, result)
                }
            }
        }
    }
}

func checkContainerStatus(group string, containers []string, nodes []types.Node) {
    printer.PrintHeader(fmt.Sprintf("Checking container status in group [%s]", group), '-')
    if nodes == nil || len(nodes) == 0 {
        printer.PrintSkipped("No host configured for [%s]", group)
        return
    }
    if containers == nil || len(containers) == 0 {
        printer.PrintSkipped("No containers configured for [%s]", group)
        return
    }

    for _, node := range nodes {
        if !util.IsNodeAddressValid(node) {
            printer.PrintErr("Current node %q has no valid address", node)
            break
        }

        printer.PrintNewLine()
        printer.Print("On node %s:", util.ToNodeLabel(node))
        cmdExecutor.SetNode(node)

        for _, container := range containers {
            cmd := fmt.Sprintf("docker ps -a -q --latest -f name=%s* | xargs --no-run-if-empty docker inspect -f '{{.State.Status}}'", container)

            sshOut, err := cmdExecutor.PerformCmd(cmd, clusterStatusOpts.Sudo)

            if err != nil {
                printer.PrintErr("Error checking status of %s: %s", container, err)
            } else {
                result := sshOut.Stdout
                if result == "running" {
                    printer.PrintOk("Container %s is running", container)
                } else if result == "created" || result == "paused" || result == "restarting" {
                    printer.PrintWarn("Container %s is %s", container, result)
                } else if result == "exited" || result == "removing" || result == "dead" {
                    printer.PrintErr("Container %s is %s", container, result)
                } else {
                    printer.PrintIgnored("Container %s not found or in unknown state: %s", container, result)
                }
            }
        }
    }
}

func checkCertificatesExpiration(group string, certificates []string, nodes []types.Node) {
    printer.PrintHeader(fmt.Sprintf("Checking certificate status in group [%s]", group), '-')
    if nodes == nil || len(nodes) == 0 {
        printer.PrintSkipped("No host configured for [%s]", group)
        return
    }
    if certificates == nil || len(certificates) == 0 {
        printer.PrintSkipped("No certificates configured for [%s]", group)
        return
    }

    for _, node := range nodes {
        if !util.IsNodeAddressValid(node) {
            printer.PrintErr("Current node %q has no valid address", node)
            break
        }

        printer.PrintNewLine()
        printer.Print("On node %s:", util.ToNodeLabel(node))
        cmdExecutor.SetNode(node)

        for _, cert := range certificates {
            cert = parseTemplate(cert, node)
            sshOut, err := cmdExecutor.PerformCmd(fmt.Sprintf("openssl x509 -enddate -noout -in %s", cert), clusterStatusOpts.Sudo)

            if err != nil {
                printer.PrintErr("Error checking expiration of %s: %s", cert, err)
            } else {
                printer.PrintOk("Certificate %s is valid until %s", cert, strings.Replace(sshOut.Stdout, "notAfter=", "", 1))
            }
        }
    }
}

func parseTemplate(value string, node types.Node) string {
    if strings.Contains(value, leftTemplateDelim) && strings.Contains(value, rightTemplateDelim) {
        printer.PrintDebug("Value containts templating. Parsing: %s", value)

        t := template.New("Template")
        t, err := t.Parse(value)
        if err != nil {
            printer.PrintDebug("Error parsing template: %s", err)
            return value
        }

        var tplResult bytes.Buffer
        err = t.Execute(&tplResult, node)
        if err != nil {
            printer.PrintDebug("Error executing template: %s", err)
        } else {
            value = tplResult.String()
            printer.PrintDebug("Template executed successfully: %s", value)
            return value
        }
    }

    printer.PrintDebug("Value does not containts templating: %s", value)

    return value
}

func checkDiskStatus(group string, diskSpace types.DiskUsage, nodes []types.Node) {
    printer.PrintHeader(fmt.Sprintf("Checking disk status in group [%s]", group), '-')
    if nodes == nil || len(nodes) == 0 {
        printer.PrintSkipped("No host configured for [%s]", group)
        return
    }

    for _, node := range nodes {
        if !util.IsNodeAddressValid(node) {
            printer.PrintErr("Current node %q has no valid address", node)
            break
        }

        printer.PrintNewLine()
        printer.Print("On node %s:", util.ToNodeLabel(node))
        cmdExecutor.SetNode(node)

        spacesRegex := regexp.MustCompile("\\s+")
        if len(diskSpace.FileSystemUsage) > 0 {
            for _, fsUsage := range diskSpace.FileSystemUsage {
                sshOut, err := cmdExecutor.PerformCmd(fmt.Sprintf("df -h | grep %s", fsUsage), clusterStatusOpts.Sudo)

                if err != nil {
                    printer.PrintErr("Error estimating file system usage for %s: %s", fsUsage, err)
                } else {
                    splits := spacesRegex.Split(sshOut.Stdout, 6)
                    fsUsed := splits[2]
                    fsAvail := splits[3]
                    fsUsePercent := strings.Replace(splits[4], "%", "", 1)
                    fsUsePercentVal, err := strconv.Atoi(fsUsePercent)

                    if err != nil {
                        printer.PrintErr("Error determining file system usage percent for %s: %s", fsUsage, err)
                    } else {
                        if fsUsePercentVal < 65 {
                            printer.PrintOk("File system usage of %s amounts to - Used: %s Available: %s (%s%s)",
                                fsUsage, fsUsed, fsAvail, fsUsePercent, "%%")
                        } else if fsUsePercentVal < 85 {
                            printer.PrintWarn("File system usage of %s amounts to - Used: %s Available: %s (%s%s)",
                                fsUsage, fsUsed, fsAvail, fsUsePercent, "%%")
                        } else {
                            printer.PrintErr("File system usage of %s amounts to - Used: %s Available: %s (%s%s)",
                                fsUsage, fsUsed, fsAvail, fsUsePercent, "%%")
                        }
                    }
                }
            }
        }

        if len(diskSpace.DirectoryUsage) > 0 {
            for _, dirUsage := range diskSpace.DirectoryUsage {
                cmd := fmt.Sprintf("du -h -d 0 %s", dirUsage)
                sshOut, err := cmdExecutor.PerformCmd(cmd, clusterStatusOpts.Sudo)

                if err != nil {
                    printer.PrintErr("Error estimating directory usage for %s: %s", dirUsage, err)
                } else {
                    splits := spacesRegex.Split(sshOut.Stdout, 2)
                    dirUse := splits[0]

                    printer.PrintOk("Directory usage of %s amounts to %s", dirUsage, dirUse)
                }
            }
        }
    }
}

func checkKubernetesStatus(group string, kubernetes types.Kubernetes, nodes []types.Node) {
    printer.PrintHeader(fmt.Sprintf("Checking Kubernetes status in group [%s]", group), '=')

    if nodes == nil || len(nodes) == 0 {
        printer.PrintCritical("No host configured for [%s]", group)
    }

    if kubernetes.Resources == nil || len(kubernetes.Resources) == 0 {
        printer.PrintCritical("No resources configured for [%s]", group)
    }

    node := ssh.GetFirstAccessibleNode(config.Ssh.LocalOn, cmdExecutor, nodes)

    if !util.IsNodeAddressValid(node) {
        printer.PrintCritical("No master available for Kubernetes status check")
    }

    printer.Print("Running kubectl on node %s\n", util.ToNodeLabel(node))

    for _, resource := range kubernetes.Resources {
        msg := fmt.Sprintf("Status of %s", resource.Type)
        namespace_msg := ""
        args := []string{"get", resource.Type}
        if resource.Namespace != "" {
            namespace_msg += " in namespace " + resource.Namespace
            args = append(args, "-n", resource.Namespace)
        }
        if resource.Wide {
            args = append(args, "-o", "wide")
        }

        printer.Print(msg + namespace_msg + ":")
        cmdExecutor.SetNode(node)
        sshOut, err := cmdExecutor.RunKubectlCommand(args)
        printer.PrintNewLine()

        if err != nil {
            printer.PrintErr("Error checking %s%s: %s", resource.Type, namespace_msg, err)
        } else {
            printer.PrintOk(sshOut.Stdout)
        }
        printer.PrintNewLine()
    }
}
