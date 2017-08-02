package types

type NetperfOpts struct {
	Output     string
	Iterations int
	Cleanup    bool
	Verbose    bool
	RootOpts
}
