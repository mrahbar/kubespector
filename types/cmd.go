package types

import (
    "github.com/mrahbar/kubernetes-inspector/integration"
)

type CommandContext struct {
    Config          Config
    Printer         integration.LogWriter
    Opts            interface{}
    CommandExecutor CommandExecutor
}

type ClusterStatusOpts struct {
    Groups string
    Checks string
    Sudo       bool
}

type GenericOpts struct {
    GroupArg  string
    NodeArg   string
    TargetArg string
    Sudo       bool
}

type ExecOpts struct {
    GenericOpts
    FileOutput string
}

type ScpOpts struct {
    GenericOpts
    LocalPath  string
    RemotePath string
}

type LogsOpts struct {
    GenericOpts
    FileOutput string
    Type       string
    Since      string
    Tail       int
    ExtraArgs  []string
}

type KubectlOpts struct {
    Command string
}
