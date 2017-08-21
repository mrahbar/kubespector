package types

import "time"

//LocalOn and Bastion are mutual exclusive
type SSHConfig struct {
	Connection SSHConnection
	LocalOn    Node
	Bastion    BastionSSHConnection
}

type SSHConnection struct {
	Username           string
	Password           string
	PrivateKey         string
	AgentAuth          bool
	Port               int
	Timeout            time.Duration
	HandshakeAttempts  int
	FileTransferMethod string
}

type BastionSSHConnection struct {
	SSHConnection
	Node Node
}

type SSHOutput struct {
	Stdout     string
	Stderr     string
	ExitStatus int
}
