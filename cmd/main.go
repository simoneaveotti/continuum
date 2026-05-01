package main

import (
	"os"

	"continuum/internal/setup"
	"continuum/internal/template"
)

func main() {
	template.SetBasePath(setup.ContinuumPath())

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	if isHelpCommand(command) {
		printUsage()
		return
	}
	if isVersionCommand(command) {
		printVersion()
		return
	}

	dispatchCommand(command, os.Args[2:])
}
