package pkg

import (
    "testing"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/stretchr/testify/assert"
    "fmt"
)

func TestStopService_Ok(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.GenericOpts{
        TargetArg: "docker",
        NodeArg: "host1",
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "systemctl stop docker", command)
        return &types.SSHOutput{}, nil
    }

    Stop(context)

    out := outBuffer.String()
    assert.True(t, called)
    assert.NotEmpty(t, out)
    assert.Contains(t, out, "Service docker stopped.")
}

func TestStopService_Error(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.GenericOpts{
        TargetArg: "docker",
        NodeArg: "host1",
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "systemctl stop docker", command)
        return &types.SSHOutput{}, fmt.Errorf("Stop failed")
    }

    Stop(context)

    out := outBuffer.String()
    assert.True(t, called)
    assert.NotEmpty(t, out)
    assert.Contains(t, out, "Error stopping service docker")
}
