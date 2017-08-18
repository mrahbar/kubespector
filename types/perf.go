package types

type NetperfOpts struct {
	OutputDir string
	Cleanup   bool
	RootOpts
}

type ScaleTestOpts struct {
	OutputDir string
	Cleanup bool
	RootOpts
}
