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

type GenericOpts struct {
    GroupArg  string
    NodeArg   string
    TargetArg string
}

type ExecOpts struct {
    GenericOpts
    Sudo       bool
    FileOutput string
}

type ScpOpts struct {
    GenericOpts
    LocalPath  string
    RemotePath string
}

type LogsOpts struct {
    GenericOpts
    Sudo       bool
    FileOutput string
    Type       string
    Since      string
    Tail       int
    ExtraArgs  []string
}

type KubectlOpts struct {
    Command string
}
