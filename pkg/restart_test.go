package pkg

import (
    "testing"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/stretchr/testify/assert"
    "fmt"
    "github.com/bouk/monkey"
    "os"
)

func TestRestartService_Ok(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.GenericOpts{
        TargetArg: "docker",
        NodeArg: "host1",
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
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
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
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
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
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

func TestRestartService_TargetMissing(t *testing.T) {
    _, outBuffer, context := defaultContext()
    context.Opts = &types.GenericOpts{
        NodeArg: "host1",
    }

    osExitCalled := false
    patch := monkey.Patch(os.Exit, func(int) {
        osExitCalled = true
    })
    defer patch.Unpatch()

    Restart(context)

    out := outBuffer.String()
    assert.True(t, osExitCalled)
    assert.NotEmpty(t, out)
    assert.Contains(t, out, "Invalid options. Parameter missing.")
}
