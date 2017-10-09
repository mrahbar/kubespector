package types

//TODO make enum of this
const ALL_GROUPNAME = "ALL"
const MASTER_GROUPNAME = "Master"
const ETCD_GROUPNAME = "Etcd"

const SERVICES_CHECKNAME = "Services"
const CONTAINERS_CHECKNAME = "Containers"
const CERTIFICATES_CHECKNAME = "Certificates"
const DISKUSAGE_CHECKNAME = "DiskUsage"
const KUBERNETES_CHECKNAME = "Kubernetes"

type Config struct {
	Ssh           SSHConfig
	ClusterGroups []ClusterGroup
}

type ClusterGroup struct {
	Name         string
	Nodes        []Node
	Services     []string
	Containers   []string
	Certificates []string
	DiskUsage    DiskUsage
	Kubernetes   Kubernetes
}

type DiskUsage struct {
	FileSystemUsage []string
	DirectoryUsage  []string
}

type Node struct {
	Host string
	IP   string
}

type Kubernetes struct {
	Resources []KubernetesResource
}

type KubernetesResource struct {
	Type      string
	Namespace string
	Wide      bool
}
