package pkg

import (
    "testing"
    "github.com/mrahbar/kubernetes-inspector/types"
    "github.com/stretchr/testify/assert"
    "github.com/bouk/monkey"
    "os"
    "io/ioutil"
    "fmt"
    "path/filepath"
    "path"
)

func TestScp_DirectionUnknown(t *testing.T) {
    _, outBuffer, context := defaultContext()
    context.Opts = &types.ScpOpts{
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "target",
        },
    }

    osExitCalled := false
    patch := monkey.Patch(os.Exit, func(int) {
        osExitCalled = true
    })
    defer patch.Unpatch()

    Scp(context)
    assert.True(t, osExitCalled)
    assert.Contains(t, outBuffer.String(), "Direction must either be 'up' or 'down' resp. first letter. Provided: 'target'")
}


func TestScp_DirectionUp_RemoteFileUnprocessable(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.ScpOpts{
        RemotePath: "/tmp/unknown",
        LocalPath: out.Name(),
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "up",
        },
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        called = true
        return &types.SSHOutput{}, fmt.Errorf("Remote lookup failed")
    }

    Scp(context)
    assert.True(t, called)
    assert.Contains(t, outBuffer.String(), "Remote path /tmp/unknown is unprocessable: Remote lookup failed")
    out.Close()
}

func TestScp_DirectionUp_LocalFileUnprocessable(t *testing.T) {
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.ScpOpts{
        RemotePath: "/tmp/known",
        LocalPath: "/tmp/unknown",
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "up",
        },
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        called = true
        return &types.SSHOutput{Stdout: "file"}, nil
    }

    Scp(context)
    assert.True(t, called)
    assert.Contains(t, outBuffer.String(), "Local path /tmp/unknown is unprocessable")
}

//---- up

func TestScp_DirectionUp_LocalDirToRemoteFile_Invalid(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    abs, _ := filepath.Abs(out.Name())
    localDir := filepath.Dir(abs)
    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.ScpOpts{
        RemotePath: "/tmp/file",
        LocalPath: filepath.Dir(abs),
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "up",
        },
    }

    called := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        called = true
        return &types.SSHOutput{Stdout: "file"}, nil
    }

    osExitCalled := false
    patch := monkey.Patch(os.Exit, func(int) {
        osExitCalled = true
    })
    defer patch.Unpatch()

    Scp(context)
    out.Close()
    assert.True(t, called)
    assert.True(t, osExitCalled)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Can not upload directory %s to remote file /tmp/file. Please choose a remote directory", localDir))
}

func TestScp_DirectionUp_LocalFileToRemoteFile_Invalid(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    mockExecutor, outBuffer, context := defaultContext()
    context.Opts = &types.ScpOpts{
        RemotePath: "/tmp/file",
        LocalPath: out.Name(),
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "up",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "file"}, nil
    }

    osExitCalled := false
    patch := monkey.Patch(os.Exit, func(int) {
        osExitCalled = true
    })
    defer patch.Unpatch()

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, osExitCalled)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Can not upload local file %s to existing remote file /tmp/file. Please choose a remote directory or a new remote filename.", out.Name()))
}


func TestScp_DirectionUp_LocalFileToRemoteDirectory_ScpError(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    mockExecutor, outBuffer, context := defaultContext()
    remote := "/tmp"
    context.Opts = &types.ScpOpts{
        RemotePath: remote,
        LocalPath:  out.Name(),
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "up",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "dir"}, nil
    }

    calledUpload := false
    errMsg := "Error uploading file"
    mockExecutor.MockUploadFile = func(remotePath string, localPath string) error {
        calledUpload = true
        return fmt.Errorf(errMsg)
    }

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, calledUpload)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Scp failed %s -> %s: %s", out.Name(), path.Join(remote, out.Name()), errMsg))
}

func TestScp_DirectionUp_LocalDirToRemoteDirectory_ScpError(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    abs, _ := filepath.Abs(out.Name())
    localDir := filepath.Dir(abs)
    mockExecutor, outBuffer, context := defaultContext()
    remoteDir := "/tmp"
    context.Opts = &types.ScpOpts{
        RemotePath: remoteDir,
        LocalPath:  localDir,
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "up",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "dir"}, nil
    }

    calledUpload := false
    errMsg := "Error uploading directory"
    mockExecutor.MockUploadDirectory = func(remotePath string, localPath string) error {
        calledUpload = true
        return fmt.Errorf(errMsg)
    }

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, calledUpload)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Scp failed %s -> %s: %s", localDir, remoteDir, errMsg))
}


func TestScp_DirectionUp_LocalFileToRemoteNoneFile_Ok(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    mockExecutor, outBuffer, context := defaultContext()
    remote := "/tmp"
    context.Opts = &types.ScpOpts{
        RemotePath: remote,
        LocalPath:  out.Name(),
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "up",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "none"}, nil
    }

    calledUpload := false
    mockExecutor.MockUploadFile = func(remotePath string, localPath string) error {
        calledUpload = true
        return nil
    }

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, calledUpload)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Scp %s -> %s finished", out.Name(), remote))
}

func TestScp_DirectionUp_LocalDirToRemoteDir_Ok(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    abs, _ := filepath.Abs(out.Name())
    localDir := filepath.Dir(abs)
    mockExecutor, outBuffer, context := defaultContext()
    remote := "/tmp"
    context.Opts = &types.ScpOpts{
        RemotePath: remote,
        LocalPath:  localDir,
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "up",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "none"}, nil
    }

    calledUpload := false
    mockExecutor.MockUploadDirectory = func(remotePath string, localPath string) error {
        calledUpload = true
        return nil
    }

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, calledUpload)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Scp %s -> %s finished", localDir, remote))
}

//---- down

func TestScp_DirectionDown_RemoteDirToLocalFile_Invalid(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    mockExecutor, outBuffer, context := defaultContext()
    remote := "/tmp"
    context.Opts = &types.ScpOpts{
        RemotePath: remote,
        LocalPath: out.Name(),
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "down",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "dir"}, nil
    }

    osExitCalled := false
    patch := monkey.Patch(os.Exit, func(int) {
        osExitCalled = true
    })
    defer patch.Unpatch()

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, osExitCalled)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Can not download remote folder %s to local file %s. Please choose a local directory.", remote, out.Name()))
}

func TestScp_DirectionDown_RemoteFileToLocalFile_Invalid(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    mockExecutor, outBuffer, context := defaultContext()
    remote := "/tmp/file"
    context.Opts = &types.ScpOpts{
        RemotePath: remote,
        LocalPath: out.Name(),
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "down",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "file"}, nil
    }

    osExitCalled := false
    patch := monkey.Patch(os.Exit, func(int) {
        osExitCalled = true
    })
    defer patch.Unpatch()

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, osExitCalled)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Can not download remote file %s to existing local file %s. Please choose a local directory or a new local filename.", remote, out.Name()))
}


func TestScp_DirectionDown_RemoteDirectoryToLocalDir_ScpError(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    abs, _ := filepath.Abs(out.Name())
    localDir := filepath.Dir(abs)
    mockExecutor, outBuffer, context := defaultContext()
    remote := "/tmp"
    context.Opts = &types.ScpOpts{
        RemotePath: remote,
        LocalPath:  localDir,
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "down",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "dir"}, nil
    }

    calledDownload := false
    errMsg := "Error downloading directory"
    mockExecutor.MockDownloadDirectory = func(remotePath string, localPath string) error {
        calledDownload = true
        return fmt.Errorf(errMsg)
    }

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, calledDownload)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Scp failed %s <- %s: %s", localDir, remote, errMsg))
}



func TestScp_DirectionDown_RemoteFileToLocalDir_ScpError(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    abs, _ := filepath.Abs(out.Name())
    localDir := filepath.Dir(abs)
    mockExecutor, outBuffer, context := defaultContext()
    remote := "/tmp/myfile"
    context.Opts = &types.ScpOpts{
        RemotePath: remote,
        LocalPath: localDir,
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "down",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "file"}, nil
    }

    calledDownload := false
    errMsg := "Error downloading file"
    mockExecutor.MockDownloadFile = func(remotePath string, localPath string) error {
        calledDownload = true
        return fmt.Errorf(errMsg)
    }

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, calledDownload)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Scp failed %s <- %s: %s", filepath.Join(localDir, "myfile"), remote, errMsg))
}

func TestScp_DirectionDown_RemoteFileToLocalDir_Ok(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    abs, _ := filepath.Abs(out.Name())
    localDir := filepath.Dir(abs)
    mockExecutor, outBuffer, context := defaultContext()
    remote := "/tmp/myfile"
    context.Opts = &types.ScpOpts{
        RemotePath: remote,
        LocalPath: localDir,
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "down",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "file"}, nil
    }

    calledDownload := false
    mockExecutor.MockDownloadFile = func(remotePath string, localPath string) error {
        calledDownload = true
        return nil
    }

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, calledDownload)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Scp %s <- %s finished", filepath.Join(localDir, "myfile"), remote))
}

func TestScp_DirectionDown_RemoteDirectoryToLocalDir_Ok(t *testing.T) {
    out, _ := ioutil.TempFile(".", "TestScp_LocalFile")
    defer os.Remove(out.Name())

    abs, _ := filepath.Abs(out.Name())
    localDir := filepath.Dir(abs)
    mockExecutor, outBuffer, context := defaultContext()
    remote := "/tmp"
    context.Opts = &types.ScpOpts{
        RemotePath: remote,
        LocalPath:  localDir,
        GenericOpts: types.GenericOpts {
            NodeArg: "host1",
            TargetArg: "down",
        },
    }

    calledCmd := false
    mockExecutor.MockPerformCmd = func(command string, sudo bool) (*types.SSHOutput, error) {
        calledCmd = true
        return &types.SSHOutput{Stdout: "dir"}, nil
    }

    calledDownload := false
    mockExecutor.MockDownloadDirectory = func(remotePath string, localPath string) error {
        calledDownload = true
        return nil
    }

    Scp(context)
    out.Close()
    assert.True(t, calledCmd)
    assert.True(t, calledDownload)
    assert.Contains(t, outBuffer.String(), fmt.Sprintf("Scp %s <- %s finished", localDir, remote))
}
