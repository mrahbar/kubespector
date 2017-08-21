package types

type Initializer func(target string, node string, group string)
type Processor func(SSHConfig, string, Node, bool)

type GenericOpts struct {
	GroupArg  string
	NodeArg   string
	TargetArg string
	RootOpts
}

type ExecOpts struct {
	GenericOpts
	Sudo       bool
	FileOutput string
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
	RootOpts
}
