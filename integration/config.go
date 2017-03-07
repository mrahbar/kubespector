package integration

type Config struct {
	Ssh SSHConfig
	Cluster struct {
		Etcd     ClusterMember
		Master   ClusterMember
		Worker   ClusterMember
		Ingress  ClusterMember
		Registry ClusterMember
	}
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

type ClusterMember struct {
	Nodes     []Node
	Services  []string
	DiskSpace DiskSpace
}

type DiskSpace struct {
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
