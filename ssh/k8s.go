package ssh

import (
	"fmt"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/mrahbar/kubernetes-inspector/types"
	"strings"
	"text/template"
	"path"
	"strconv"
)

func RunKubectlCommand(sshOpts types.SSHConfig, node types.Node, args []string, debug bool) (*types.SSHOutput, error) {
	a := strings.Join(args, " ")
	return PerformCmd(sshOpts, node, fmt.Sprintf("kubectl %s", a), debug)
}

func DeployKubernetesResource(sshOpts types.SSHConfig, node types.Node, tpl string, data interface{}, debug bool) (*types.SSHOutput, error) {
	var definition bytes.Buffer

	tmpl, _ := template.New("kube-template").Parse(tpl)
	tmpl.Execute(&definition, data)

	tmpFile, err := ioutil.TempFile("", "kubespector-")
	if err != nil {
		return &types.SSHOutput{}, err
	}

	defer os.Remove(tmpFile.Name())
	ioutil.WriteFile(tmpFile.Name(), definition.Bytes(), os.ModeAppend)
	remoteFile := path.Join("/tmp", filepath.Base(tmpFile.Name()))

	err = UploadFile(sshOpts, node, remoteFile, tmpFile.Name(), debug)
	if err != nil {
		return &types.SSHOutput{}, err
	}

	args := []string{"apply", "-f", remoteFile}
	result, err := RunKubectlCommand(sshOpts, node, args, debug)
	DeleteRemoteFile(sshOpts, node, remoteFile, debug)

	return result, err
}

func GetNumberOfReadyNodes(sshOpts types.SSHConfig, node types.Node, debug bool) (int, error) {
	tmpl := "\"{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}\""
	args := []string{"get", "nodes", "-o", "jsonpath=" + tmpl, " | ", "tr", "';'", "\\\\n", " | ", "grep", "\"Ready=True\"", " | ", "wc", "-l"}
	sshOut, err := RunKubectlCommand(sshOpts, node, args, debug)

	if err != nil {
		return -1, err
	} else {
		count, errAtoi := strconv.Atoi(strings.TrimRight(sshOut.Stdout, "\n"))

		if errAtoi != nil {
			return -1, err
		} else {
			return count, err
		}
	}
}

func CreateNamespace(sshOpts types.SSHConfig, node types.Node, namespace string, debug bool) error {
	data := make(map[string]string)
	data["Namespace"] = namespace
	_, err := DeployKubernetesResource(sshOpts, node, types.NAMESPACE_TEMPLATE, data, debug)

	return err
}

func CreateService(sshOpts types.SSHConfig, node types.Node, serviceData interface{}, debug bool) (bool, error) {
	sshOut, err := DeployKubernetesResource(sshOpts, node, types.SERVICE_TEMPLATE, serviceData, debug)

	if err != nil {
		if strings.Contains(CombineOutput(sshOut), "AlreadyExists") {
			return true, nil
		} else {
			return false, err
		}
	}

	return false, nil
}

func CreateReplicationController(sshOpts types.SSHConfig, node types.Node, data interface{}, debug bool) error {
	_, err := DeployKubernetesResource(sshOpts, node, types.REPLICATION_CONTROLLER_TEMPLATE, data, debug)
	return err
}

func GetPods(sshOpts types.SSHConfig, node types.Node, namespace string, wide bool, debug bool) (*types.SSHOutput, error) {
	args := []string{"--namespace=" + namespace, "get", "pods"}
	if wide {
		args = append(args, "-o=wide")
	}

	return RunKubectlCommand(sshOpts, node, args, debug)
}

func RemoveResource(sshOpts types.SSHConfig, node types.Node, namespace, full_qualified_name string, debug bool) error {
	args := []string{"--namespace=" + namespace, "delete", full_qualified_name}
	_, err := RunKubectlCommand(sshOpts, node, args, debug)

	return err
}
