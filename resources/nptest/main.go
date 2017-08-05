package main

/*
 Dual-mode program - runs as both the orchestrator and as the worker nodes depending on command line flags
 The RPC API is contained wholly within this file.
*/

import (
	"flag"
	"github.com/mrahbar/kubernetes-inspector/resources/nptest/integration"
	"github.com/mrahbar/kubernetes-inspector/resources/nptest/pkg"
	"os"
)

var mode string
var debug bool

func init() {
	flag.StringVar(&mode, "mode", "worker", "Mode for the daemon (worker | orchestrator)")
	flag.BoolVar(&debug, "debug", false, "Increase debugging output")
}

func main() {
	initializeOutputFiles()
	flag.Parse()
	if !validateParams() {
		integration.PrettyPrintErr("Failed to parse cmdline args - fatal error - bailing out")
		os.Exit(1)
	}

	integration.PrintHeader("Running as "+mode+" ", '=')
	if mode == pkg.OrchestratorMode {
		pkg.Orchestrate(debug)
	} else {
		pkg.Work(debug)
	}
	integration.PrettyPrint("Terminating")
}

func initializeOutputFiles() {
	fd, err := os.OpenFile(pkg.OutputCaptureFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		integration.PrettyPrintErr("Failed to open output capture file: %s", err)
		os.Exit(2)
	}
	fd.Close()
}

func validateParams() (rv bool) {
	rv = true
	if mode != pkg.WorkerMode && mode != pkg.OrchestratorMode {
		integration.PrettyPrintErr("Invalid mode", mode)
		return false
	}

	port := os.Getenv(pkg.EnvOrchestratorPort)
	if mode == pkg.WorkerMode && len(port) == 0 {
		integration.PrettyPrintErr("Invalid %s", pkg.EnvOrchestratorPort, port)
		return false
	}

	address := os.Getenv(pkg.EnvOrchestratorPodIP)
	if mode == pkg.WorkerMode && len(address) == 0 {
		integration.PrettyPrintErr("Invalid %s", pkg.EnvOrchestratorPodIP, address)
		return false
	}

	return
}
