package pkg

import (
    "testing"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/stretchr/testify/assert"
    "fmt"
    "github.com/bouk/monkey"
    "os"
)

func TestKubectlService_NoMasters(t *testing.T) {
    _, outBuffer, context := defaultContext()
    context.Opts = &types.KubectlOpts{
        Command: "version",
    }

    osExitCalled := false
    patch := monkey.Patch(os.Exit, func(int) {
        osExitCalled = true
    })
    defer patch.Unpatch()

    context.Config.ClusterGroups = []types.ClusterGroup{}
    Kubectl(context)
    assert.True(t, osExitCalled)
    assert.Contains(t, outBuffer.String(), "No host configured for group [Master]")
}

func TestKubectl_SecondNodeAccessible(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.KubectlOpts{
        Command: "version",
    }

    calledTimes := 0
    kubectlVersion := "kubectl version v1.7.3"
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        if command == "hostname" {
            calledTimes++
            if mockExecutor.Node.Host == "host1" {
                return &types.SSHOutput{}, fmt.Errorf("SSH failed")
            } else {
                return &types.SSHOutput{}, nil
            }
        }

        return &types.SSHOutput{}, nil
    }

    mockExecutor.MockRunKubectlCommand = func(args []string) (*types.SSHOutput, error) {
        if args[0] == "version" {
            calledTimes++
            return &types.SSHOutput{Stdout: kubectlVersion}, nil
        }

        return &types.SSHOutput{}, nil
    }

    Kubectl(context)
    out := outBuffer.String()
    assert.Equal(t, calledTimes, 3)
    assert.NotEmpty(t, out)
    assert.Equal(t, "host3", mockExecutor.Node.Host)
    assert.Contains(t, out, kubectlVersion)
}

func TestKubectl_Error(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.KubectlOpts{
        Command: "version",
    }

    called := false
    kubectlVersion := "Api server down"
    mockExecutor.MockRunKubectlCommand = func(args []string) (*types.SSHOutput, error) {
        if args[0] == "version" {
            called = true
            return &types.SSHOutput{}, fmt.Errorf(kubectlVersion)
        }

        return &types.SSHOutput{}, nil
    }

    Kubectl(context)
    out := outBuffer.String()
    assert.True(t, called)
    assert.NotEmpty(t, out)
    assert.Contains(t, out, "Error performing kubectl command 'version': "+kubectlVersion)
}

