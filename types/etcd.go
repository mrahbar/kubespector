package types

type RootOpts struct {
	Debug bool
}

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
	EtcdOpts
	RootOpts
}
