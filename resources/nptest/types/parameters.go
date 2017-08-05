package types

type Orchestrator struct {
	Port    string
	Address string
}

type Worker struct {
	Node    string
	Worker  string
	Address string
}
