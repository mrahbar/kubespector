package pkg

import (
    "testing"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/stretchr/testify/assert"
    "github.com/bouk/monkey"
    "os"
    "io/ioutil"
)

func TestLogs_UnknownType(t *testing.T) {
    _, outBuffer, context := defaultContext()
    context.Opts = &types.LogsOpts{
        Type: "unknown",
        GenericOpts: types.GenericOpts{
            NodeArg: "host1",
        },
    }

    osExitCalled := false
    patch := monkey.Patch(os.Exit, func(int) {
        osExitCalled = true
    })
    defer patch.Unpatch()
    Logs(context)
    assert.True(t, osExitCalled)
    assert.Contains(t, outBuffer.String(), "Unknown type unknown")
}

func TestLogs_Service(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.LogsOpts{
        Type: "service",
        Since: "10m",
        Tail: 10,
        Sudo: true,
        GenericOpts: types.GenericOpts{
            NodeArg: "host1",
            TargetArg: "kubelet",
        },
    }

    called := false
    logsOut := "Kubelet logs"
    mockExecutor.MockPerformCmd = func(command string) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "sudo journalctl --lines=10 --since=10m --unit=kubelet", command)
        return &types.SSHOutput{Stdout: logsOut}, nil
    }

    Logs(context)
    assert.True(t, called)
    assert.Contains(t, outBuffer.String(), logsOut)
}

func TestLogs_Docker(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.LogsOpts{
        Type: "container",
        Since: "10m",
        Tail: 10,
        Sudo: true,
        GenericOpts: types.GenericOpts{
            NodeArg: "host1",
            TargetArg: "kubelet",
        },
    }

    called := false
    logsOut := "Kubelet logs"
    mockExecutor.MockPerformCmd = func(command string) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "sudo docker logs --tail 10 --since 10m kubelet", command)
        return &types.SSHOutput{Stdout: logsOut}, nil
    }

    Logs(context)
    assert.True(t, called)
    assert.Contains(t, outBuffer.String(), logsOut)
}

func TestLogs_Pod(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.LogsOpts{
        Type: "pod",
        Since: "10m",
        Tail: 10,
        Sudo: false,
        GenericOpts: types.GenericOpts{
            NodeArg: "host1",
            TargetArg: "kube-dns",
        },
    }

    called := false
    logsOut := "kube-dns logs"
    mockExecutor.MockPerformCmd = func(command string) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "kubectl logs --tail=10 --since=10m kube-dns", command)
        return &types.SSHOutput{Stdout: logsOut}, nil
    }

    Logs(context)
    assert.True(t, called)
    assert.Contains(t, outBuffer.String(), logsOut)
}

func TestLogs_FileOutput(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestLogs_FileOutput")
    defer os.Remove(out.Name())
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.LogsOpts{
        Type: "pod",
        Since: "10m",
        Tail: 10,
        Sudo: false,
        FileOutput: out.Name(),
        GenericOpts: types.GenericOpts{
            NodeArg: "host1",
            TargetArg: "kube-dns",
        },
    }

    called := false
    logsOut := "kube-dns logs"
    mockExecutor.MockPerformCmd = func(command string) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "kubectl logs --tail=10 --since=10m kube-dns", command)
        return &types.SSHOutput{Stdout: logsOut}, nil
    }

    Logs(context)
    assert.True(t, called)
    assert.NotContains(t, outBuffer.String(), logsOut)

    b, _ := ioutil.ReadFile(out.Name())
    assert.Contains(t, string(b), logsOut)
    out.Close()
}

