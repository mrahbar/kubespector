package pkg

import (
    "testing"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/stretchr/testify/assert"
    "fmt"
)

func TestRestartService_Ok(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.GenericOpts{
        TargetArg: "docker",
        NodeArg: "host1",
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "systemctl restart docker", command)
        return &types.SSHOutput{}, nil
    }

    Restart(context)
    assert.True(t, called)
    assert.Contains(t, outBuffer.String(), "Service docker restarted.")
}

func TestRestartServiceMultipleNodes_Ok(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.GenericOpts{
        TargetArg: "docker",
        NodeArg: "host1,host2",
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "systemctl restart docker", command)
        return &types.SSHOutput{}, nil
    }

    Restart(context)
    assert.True(t, called)
    assert.Contains(t, outBuffer.String(), "Result on node host1")
    assert.Contains(t, outBuffer.String(), "Result on node host2")
}

func TestRestartService_Error(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.GenericOpts{
        TargetArg: "docker",
        NodeArg: "host1",
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "systemctl restart docker", command)
        return &types.SSHOutput{}, fmt.Errorf("Restart failed")
    }

    Restart(context)

    out := outBuffer.String()
    assert.True(t, called)
    assert.NotEmpty(t, out)
    assert.Contains(t, out, "Error restarting service docker: Restart failed", )
}
