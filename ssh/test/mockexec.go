package test

import (
    "github.com/mrahbar/kubernetes-inspector/types"
)

// ******************** MockExecutor ********************
type MockExecutor struct {
    Node types.Node
    MockSetNode    func(node types.Node)
    MockGetNode    func() types.Node
    MockPerformCmd func(command string, sudo bool) (*types.SSHOutput, error)

    MockDownloadFile      func(remotePath string, localPath string) error
    MockDownloadDirectory func(remotePath string, localPath string) error
    MockUploadFile        func(remotePath string, localPath string) error
    MockUploadDirectory   func(remotePath string, localPath string) error
    MockDeleteRemoteFile  func(remoteFile string) error

    MockRunKubectlCommand        func(args []string) (*types.SSHOutput, error)
    MockDeployKubernetesResource func(tpl string, data interface{}) (*types.SSHOutput, error)

    MockGetNumberOfReadyNodes       func() (int, error)
    MockCreateNamespace             func(namespace string) error
    MockCreateService               func(serviceData interface{}) (bool, error)
    MockCreateReplicationController func(data interface{}) error
    MockScaleReplicationController  func(namespace string, rc string, replicas int) error
    MockGetPods                     func(namespace string, wide bool) (*types.SSHOutput, error)
    MockRemoveResource              func(namespace, fullQualifiedName string) error
}

func (e *MockExecutor) GetNode() types.Node {
    if e.MockGetNode != nil {
        return e.MockGetNode()
    } else {
        return e.Node
    }
}

func (e *MockExecutor) SetNode(node types.Node) {
    if e.MockSetNode != nil {
        e.MockSetNode(node)
    } else {
        e.Node = node
    }
}

func (e *MockExecutor) PerformCmd(command string, sudo bool) (*types.SSHOutput, error) {
    if e.MockPerformCmd != nil {
        return e.MockPerformCmd(command, sudo)
    }

    return &types.SSHOutput{}, nil
}

func (e *MockExecutor) DownloadFile(remotePath string, localPath string) error {
    if e.MockDownloadFile != nil {
        return e.MockDownloadFile(remotePath, localPath)
    }

    return nil
}

func (e *MockExecutor) DownloadDirectory(remotePath string, localPath string) error {
    if e.MockDownloadDirectory != nil {
        return e.MockDownloadDirectory(remotePath, localPath)
    }

    return nil
}

func (e *MockExecutor) UploadFile(remotePath string, localPath string) error {
    if e.MockUploadFile != nil {
        return e.MockUploadFile(remotePath, localPath)
    }

    return nil
}

func (e *MockExecutor) UploadDirectory(remotePath string, localPath string) error {
    if e.MockUploadDirectory != nil {
        return e.MockUploadDirectory(remotePath, localPath)
    }

    return nil
}

func (e *MockExecutor) DeleteRemoteFile(remoteFile string) error {
    if e.MockDeleteRemoteFile != nil {
        return e.MockDeleteRemoteFile(remoteFile)
    }

    return nil
}

func (e *MockExecutor) RunKubectlCommand(args []string) (*types.SSHOutput, error) {
    if e.MockRunKubectlCommand != nil {
        return e.MockRunKubectlCommand(args)
    }

    return &types.SSHOutput{}, nil
}

func (e *MockExecutor) DeployKubernetesResource(tpl string, data interface{}) (*types.SSHOutput, error) {
    if e.MockDeployKubernetesResource != nil {
        return e.MockDeployKubernetesResource(tpl, data)
    }

    return &types.SSHOutput{}, nil
}

func (e *MockExecutor) GetNumberOfReadyNodes() (int, error) {
    if e.MockGetNumberOfReadyNodes != nil {
        return e.MockGetNumberOfReadyNodes()
    }

    return -1, nil
}

func (e *MockExecutor) CreateNamespace(namespace string) error {
    if e.MockCreateNamespace != nil {
        return e.MockCreateNamespace(namespace)
    }

    return nil
}

func (e *MockExecutor) CreateService(serviceData interface{}) (bool, error) {
    if e.MockCreateService != nil {
        return e.MockCreateService(serviceData)
    }

    return false, nil
}

func (e *MockExecutor) CreateReplicationController(data interface{}) error {
    if e.MockCreateReplicationController != nil {
        return e.MockCreateReplicationController(data)
    }

    return nil
}

func (e *MockExecutor) ScaleReplicationController(namespace string, rc string, replicas int) error {
    if e.MockScaleReplicationController != nil {
        return e.MockScaleReplicationController(namespace, rc, replicas)
    }

    return nil
}

func (e *MockExecutor) GetPods(namespace string, wide bool) (*types.SSHOutput, error) {
    if e.MockGetPods != nil {
        return e.MockGetPods(namespace, wide)
    }

    return &types.SSHOutput{}, nil
}

func (e *MockExecutor) RemoveResource(namespace string, fullQualifiedName string) error {
    if e.MockRemoveResource != nil {
        return e.MockRemoveResource(namespace, fullQualifiedName)
    }

    return nil
}
