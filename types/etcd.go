package types

type EtcdOpts struct {
	ClientCertAuth bool
	Endpoint       string
	CaFile         string
	ClientCertFile string
	ClientKeyFile  string
}

type EtcdBackupOpts struct {
	Output  string
	DataDir string
	Sudo       bool
	EtcdOpts
}
