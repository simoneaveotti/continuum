package main

import "errors"

var agentInstallUsage = "Usage: ctx agent install [--project=<name>] [--force]"
var agentRemoveUsage = "Usage: ctx agent remove [--project=<name>]"

func parseAgentInstallArgs(args []string) (projectName string, force bool) {
	for _, arg := range args {
		if arg == "--force" {
			force = true
		} else if val, ok := parseFlag(arg, "--project="); ok {
			projectName = val
		}
	}
	return projectName, force
}

var agentProjectUsage = "Usage: ctx agent status [--project=<name>]\n       ctx agent update [--project=<name>] [--force]"
var agentUsage = []string{"Usage: ctx agent <command> [options]", "Commands: status, update, install", "  ctx agent status [--project=<name>]", "  ctx agent update [--project=<name>] [--force]"}

func parseAgentProjectArgs(args []string) (projectName string, force bool, err error) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			projectName = val
			continue
		}
		if arg == "--force" {
			force = true
			continue
		}
		return "", false, errors.New(agentProjectUsage)
	}
	return projectName, force, nil
}
