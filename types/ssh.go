package types

import "time"

//LocalOn and Bastion are mutual exclusive
type SSHConfig struct {
	Connection SSHConnection
	LocalOn    Node
	Bastion    BastionSSHConnection
}

type SSHConnection struct {
	Username           string
	Password           string
	PrivateKey         string
	AgentAuth          bool
	Port               int
	Timeout            time.Duration
	HandshakeAttempts  int
	FileTransferMethod string
}

type BastionSSHConnection struct {
	SSHConnection
	Node Node
}

type SSHOutput struct {
	Stdout     string
	Stderr     string
	ExitStatus int
}

type CommandExecutor interface {
    SetNode(node Node)
    PerformCmd(command string) (*SSHOutput, error)

    DownloadFile(remotePath string, localPath string) error
    DownloadDirectory(remotePath string, localPath string) error
    UploadFile(remotePath string, localPath string) error
    UploadDirectory(remotePath string, localPath string) error
    DeleteRemoteFile(remoteFile string) error

    RunKubectlCommand(args []string) (*SSHOutput, error)
    DeployKubernetesResource(tpl string, data interface{}) (*SSHOutput, error)

    GetNumberOfReadyNodes() (int, error)
    CreateNamespace(namespace string) error
    CreateService(serviceData interface{}) (bool, error)
    CreateReplicationController(data interface{}) error
    ScaleReplicationController(namespace string, rc string, replicas int) error
    GetPods(namespace string, wide bool) (*SSHOutput, error)
    RemoveResource(namespace, fullQualifiedName string) error
}