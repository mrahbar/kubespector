package pkg

import (
    "testing"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/stretchr/testify/assert"
    "github.com/bouk/monkey"
    "os"
    "fmt"
    "strings"
)

func TestExec_CommandMissing(t *testing.T) {
    _, outBuffer, context := defaultContext()
    context.Opts = &types.ExecOpts{
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
        },
    }

    osExitCalled := false
    patch := monkey.Patch(os.Exit, func(int) {
        osExitCalled = true
    })
    defer patch.Unpatch()

    Exec(context)

    out := outBuffer.String()
    assert.True(t, osExitCalled)
    assert.NotEmpty(t, out)
    assert.Contains(t, out, "Invalid options. Parameter missing.")
}


func TestExec_CmdFailed(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.ExecOpts{
        GenericOpts: types.GenericOpts {
            TargetArg: "pwd",
            NodeArg: "host1",
        },
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        called = true
        return &types.SSHOutput{}, fmt.Errorf("pwd failed")
    }

    Exec(context)

    out := outBuffer.String()
    assert.True(t, called)
    assert.NotEmpty(t, out)
    assert.Contains(t, out, "Error executing command: pwd failed")
}

func TestExec_CmdOk(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.ExecOpts{
        GenericOpts: types.GenericOpts {
            TargetArg: "pwd",
            NodeArg: "host1",
        },
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        called = true
        return &types.SSHOutput{Stdout:"~"}, nil
    }

    Exec(context)

    out := outBuffer.String()
    assert.True(t, called)
    assert.NotEmpty(t, out)
    assert.Contains(t, out, "~")
}

func TestExec_OrderHosts(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.ExecOpts{
        GenericOpts: types.GenericOpts {
            TargetArg: "pwd",
            NodeArg: "host3,host1",
        },
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        called = true
        return &types.SSHOutput{Stdout:"~"}, nil
    }

    Exec(context)

    out := outBuffer.String()
    assert.True(t, called)
    assert.NotEmpty(t, out)
    assert.True(t, strings.Index(out, "host1") < strings.Index(out, "host3"))
}
