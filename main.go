package main

import "github.com/mrahbar/kubernetes-inspector/cmd"

// Set via linker flag
var version string
var buildDate string

func main() {
	cmd.Execute(version, buildDate)
}
