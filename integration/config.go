package integration

type Config struct {
	Ssh SSHConfig
	Cluster struct{
		Etcd ClusterMember
		Master ClusterMember
		Worker ClusterMember
	}
	Kubernetes struct{
		Resources []KubernetesResource
	}
}

type SSHConfig struct {
	Pty  bool
	User string
	Key  string
	Port int
	Options  string
}

type ClusterMember struct {
	Nodes []Node
	Services []string
	Directories []string
}

type Node struct {
	Host string
	IP   string
}

type KubernetesResource struct {
	Type string
	Namespace string
	Wide bool
}
