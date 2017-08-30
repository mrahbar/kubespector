package main

import "github.com/mrahbar/kubernetes-inspector/cmd"

// Set via linker flag
var version string
var buildDate string
var branch string
var commit string

func main() {
	cmd.Execute(cmd.BuildInformation{Version: version, BuildDate: buildDate, Branch: branch, Commit: commit})
}
