package main

import (
	"errors"
	"fmt"
	"strings"
)

var projectUsage = []string{"Usage: ctx project <command> [options]", "Commands: list, delete, onboard, init"}

func parseProjectCommandArgs(args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("expected exactly one project name")
	}
	if strings.HasPrefix(args[0], "--") {
		return "", fmt.Errorf("project commands accept a positional project name, not flags")
	}
	return args[0], nil
}
func parseProjectDeleteArgs(args []string) (project string, autoConfirm bool, err error) {
	for _, arg := range args {
		if arg == "--yes" {
			autoConfirm = true
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return "", false, fmt.Errorf("unknown flag: %s", arg)
		}
		if project != "" {
			return "", false, fmt.Errorf("expected exactly one project name")
		}
		project = arg
	}
	if project == "" {
		return "", false, fmt.Errorf("expected exactly one project name")
	}
	return project, autoConfirm, nil
}

var projectOnboardUsage = "Usage: ctx project onboard <project> [--force] [--yes]"

func parseProjectOnboardArgs(args []string) (project string, force bool, autoConfirm bool, err error) {
	for _, arg := range args {
		if arg == "--force" {
			force = true
		} else if arg == "--yes" {
			autoConfirm = true
		} else if strings.HasPrefix(arg, "--") {
			return "", false, false, errors.New(projectOnboardUsage)
		} else if project == "" {
			project = arg
		} else {
			return "", false, false, errors.New(projectOnboardUsage)
		}
	}
	if project == "" {
		return "", false, false, errors.New(projectOnboardUsage)
	}
	return project, force, autoConfirm, nil
}
