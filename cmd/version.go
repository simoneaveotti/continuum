package main

import (
	"fmt"
	"runtime/debug"
)

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
	fmt.Printf("ctx %s\n", versionString())
}

func versionString() string {
	if version != "" && version != "dev" {
		return version
	}
	if commit != "" && commit != "none" {
		return "dev+" + shortCommit(commit)
	}
	if revision, modified := vcsBuildInfo(); revision != "" {
		suffix := ""
		if modified {
			suffix = "-dirty"
		}
		return "dev+" + shortCommit(revision) + suffix
	}
	if version != "" {
		return version
	}
	return "dev"
}

func vcsBuildInfo() (revision string, modified bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	return revision, modified
}

func shortCommit(value string) string {
	if len(value) <= 7 {
		return value
	}
	return value[:7]
}
