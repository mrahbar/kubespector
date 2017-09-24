package types

type NetperfOpts struct {
	OutputDir string
	Cleanup   bool
}

type ScaleTestOpts struct {
	OutputDir string
	MaxReplicas int
	Cleanup   bool
}
