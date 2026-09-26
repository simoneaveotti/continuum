package main

import (
	"os"

	"continuum/internal/setup"
	"continuum/internal/template"
)

func main() {
	template.SetBasePath(setup.ContinuumPath())

	if len(os.Args) < 2 {
		printUsage(false)
		return
	}

	command := os.Args[1]
	if isHelpCommand(command) {
		verbose, err := parseHelpArgs(os.Args[2:])
		if err != nil {
			dieUsage(err.Error())
		}
		printUsage(verbose)
		return
	}
	if isVersionCommand(command) {
		printVersion()
		return
	}

	dispatchCommand(command, os.Args[2:])
}
