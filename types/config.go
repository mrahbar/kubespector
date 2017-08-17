package types

//TODO make enum of this
const KUBERNETES_GROUPNAME = "Kubernetes"
const MASTER_GROUPNAME = "Master"
const ETCD_GROUPNAME = "Etcd"

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

//LocalOn and Bastion are mutual exclusive
type SSHConfig struct {
	Connection SSHConnection
	LocalOn    Node
	Bastion    BastionSSHConnection
}

type SSHConnection struct {
	User    string
	Key     string
	Port    int
}

type BastionSSHConnection struct {
	SSHConnection
	Node Node
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
