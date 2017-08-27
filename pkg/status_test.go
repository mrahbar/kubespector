package pkg

import (
    "testing"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/stretchr/testify/assert"
    "fmt"
)

func TestStatusService_Ok(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.GenericOpts{
        TargetArg: "docker",
        NodeArg: "host1",
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "sudo systemctl status docker -l", command)
        return &types.SSHOutput{Stdout: "TEST OK"}, nil
    }

    Status(context)
    assert.True(t, called)
    assert.Contains(t, outBuffer.String(), "TEST OK")
}

func TestStatusService_Error(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.GenericOpts{
        TargetArg: "docker",
        NodeArg: "host1",
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string) (*types.SSHOutput, error) {
        called = true
        assert.Equal(t, "sudo systemctl status docker -l", command)
        return &types.SSHOutput{}, fmt.Errorf("Status failed")
    }

    Status(context)

    out := outBuffer.String()
    assert.True(t, called)
    assert.NotEmpty(t, out)
    assert.Contains(t, out, "Error checking status of service docker: Status failed", )
}
