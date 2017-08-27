package pkg

import (
    "github.com/mrahbar/kubernetes-inspector/types"
    "time"
    "bytes"
    printTest "github.com/mrahbar/kubernetes-inspector/integration/test"
    sshTest "github.com/mrahbar/kubernetes-inspector/ssh/test"
)

func defaultContext() (*sshTest.MockExecutor, *bytes.Buffer, *types.CommandContext) {
    mockExecutor := &sshTest.MockExecutor{}
    buf := &bytes.Buffer{}
    context := &types.CommandContext{
        Printer: &printTest.MockLogWriter{
            Out: buf,
        },
        Config: types.Config{
            Ssh: types.SSHConfig{
                Connection: types.SSHConnection{
                    Username:   "testuser",
                    Port:       22,
                    PrivateKey: "./private.key",
                    Timeout:    time.Second,
                },
            },
            ClusterGroups: []types.ClusterGroup{
                {
                    Name: types.MASTER_GROUPNAME,
                    Nodes: []types.Node{
                        {Host: "host1",},
                        {Host: "host2",},
                    },
                    Certificates: []string{"/etc/kubernetes/certs/ca.pem"},
                    Containers:   []string{"k8s_kube-apiserver", "k8s_kube-controller-manager", "k8s_kube-scheduler"},
                    Services:     []string{"docker"},
                    DiskUsage: types.DiskUsage{
                        DirectoryUsage:  []string{"/var/log"},
                        FileSystemUsage: []string{"/dev/sda1"},
                    },
                },
            },
        },
        CommandExecutor: mockExecutor,
    }

    return mockExecutor, buf, context
}
