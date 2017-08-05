package pkg

var debug bool

// Worker specific
const (
	WorkerMode        = "worker"
	iperf3Path        = "/usr/bin/iperf3"
	netperfPath       = "/usr/local/bin/netperf"
	netperfServerPath = "/usr/local/bin/netserver"
	parallelStreams   = "8"
)

// Orchestrator specific
const (
	OrchestratorMode  = "orchestrator"
	OutputCaptureFile = "/tmp/output.txt"
	mssMin            = 96
	mssMax            = 1460
	mssStepSize       = 64

	RpcServicePort = "5202"
	iperf3Port     = "5201"
	netperfPort    = "12865"

	netperf_w2_service_host = "netperf-w2"

	csvDataMarker    = "GENERATING CSV OUTPUT"
	csvEndDataMarker = "END CSV DATA"
)

const (
	iperfTcpTest = iota
	iperfUdpTest = iota
	netperfTest  = iota
)
