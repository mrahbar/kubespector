package integration

const KUBERNETES_GROUPNAME = "Kubernetes"
const MASTER_GROUPNAME = "Master"

const SERVICES_CHECKNAME = "Services"
const CONTAINERS_CHECKNAME = "Containers"
const CERTIFICATES_CHECKNAME = "Certificates"
const DISKUSAGE_CHECKNAME = "DiskUsage"

type Config struct {
	Ssh           SSHConfig
	ClusterGroups []ClusterGroup
	Kubernetes struct {
		Resources []KubernetesResource
	}
}

type SSHConfig struct {
	User    string
	Key     string
	Port    int
	Pty     bool
	Sudo    bool
	Options string
}

type ClusterGroup struct {
	Name         string
	Nodes        []Node
	Services     []string
	Containers   []string
	Certificates []string
	DiskUsage    DiskUsage
}

type DiskUsage struct {
	FileSystemUsage []string
	DirectoryUsage  []string
}

type Node struct {
	Host string
	IP   string
}

type KubernetesResource struct {
	Type      string
	Namespace string
	Wide      bool
}
