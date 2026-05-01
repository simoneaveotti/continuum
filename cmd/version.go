package main

import "fmt"

var version = "dev"
var commit = "none"
var date = "unknown"

func isHelpCommand(command string) bool {
	return command == "--help" || command == "-h"
}

func isVersionCommand(command string) bool {
	return command == "--version" || command == "version"
}

func printVersion() {
	fmt.Printf("ctx %s\n", version)
}
