package ssh

import (
	"bytes"
    "fmt"
    "github.com/mrahbar/kubernetes-inspector/types"
	"io/ioutil"
	"os"
    "path"
	"path/filepath"
    "strconv"
	"strings"
	"text/template"
)

func (c *Executor) RunKubectlCommand(args []string) (*types.SSHOutput, error) {
	a := strings.Join(args, " ")
    return c.PerformCmd(fmt.Sprintf("kubectl %s", a))
}

func (c *Executor) DeployKubernetesResource(tpl string, data interface{}) (*types.SSHOutput, error) {
	var definition bytes.Buffer

	tmpl, _ := template.New("kube-template").Parse(tpl)
	tmpl.Execute(&definition, data)

    c.Printer.PrintTrace("Generated template:\n%s", definition.String())

	tmpFile, err := ioutil.TempFile("", "kubespector-")
	if err != nil {
		return &types.SSHOutput{}, err
	}

	defer os.Remove(tmpFile.Name())
	ioutil.WriteFile(tmpFile.Name(), definition.Bytes(), os.ModeAppend)
	remoteFile := path.Join("/tmp", filepath.Base(tmpFile.Name()))

    err = c.UploadFile(remoteFile, tmpFile.Name())
	if err != nil {
		return &types.SSHOutput{}, err
	}

	args := []string{"apply", "-f", remoteFile}
    result, err := c.RunKubectlCommand(args)
    c.DeleteRemoteFile(remoteFile)

	return result, err
}

func (c *Executor) GetNumberOfReadyNodes() (int, error) {
	tmpl := "\"{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}\""
	args := []string{"get", "nodes", "-o", "jsonpath=" + tmpl, " | ", "tr", "';'", "\\\\n", " | ", "grep", "\"Ready=True\"", " | ", "wc", "-l"}
    sshOut, err := c.RunKubectlCommand(args)

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

func (c *Executor) CreateNamespace(namespace string) error {
	data := make(map[string]string)
	data["Namespace"] = namespace
    _, err := c.DeployKubernetesResource(types.NAMESPACE_TEMPLATE, data)

	return err
}

func (c *Executor) CreateService(serviceData interface{}) (bool, error) {
    sshOut, err := c.DeployKubernetesResource(types.SERVICE_TEMPLATE, serviceData)

	if err != nil {
		if strings.Contains(CombineOutput(sshOut), "AlreadyExists") {
			return true, nil
		} else {
			return false, err
		}
	}

	return false, nil
}

func (c *Executor) CreateReplicationController(data interface{}) error {
    _, err := c.DeployKubernetesResource(types.REPLICATION_CONTROLLER_TEMPLATE, data)
	return err
}

func (c *Executor) ScaleReplicationController(namespace string, rc string, replicas int) error {
    args := []string{"--namespace=" + namespace, "scale", "replicationcontroller", rc, fmt.Sprintf("--replicas=%d", replicas)}
    _, err := c.RunKubectlCommand(args)
    return err
}

func (c *Executor) GetPods(namespace string, wide bool) (*types.SSHOutput, error) {
	args := []string{"--namespace=" + namespace, "get", "pods"}
	if wide {
		args = append(args, "-o=wide")
	}

    return c.RunKubectlCommand(args)
}

func (c *Executor) RemoveResource(namespace, fullQualifiedName string) error {
    args := []string{"--namespace=" + namespace, "delete", fullQualifiedName}
    _, err := c.RunKubectlCommand(args)

	return err
}
