package main

import (
	"fmt"
	"strconv"
	"strings"
)

var captureUsage = "Usage: ctx capture <task> --project=<name> [--type=state|proposal|request|response|decision] [--resolves=<filename>] [--yes]"

func parseCaptureArgs(args []string) (taskName, project, captureType, resolves string, autoConfirm bool) {
	captureType = "state"
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if val, ok := parseFlag(arg, "--type="); ok {
			captureType = val
		} else if val, ok := parseFlag(arg, "--resolves="); ok {
			resolves = val
		} else if arg == "--yes" {
			autoConfirm = true
		} else if taskName == "" {
			taskName = arg
		}
	}
	return taskName, project, captureType, resolves, autoConfirm
}

var handoffUsage = "Usage: ctx handoff <task> [--project=<name>] [--yes]"

func parseTaskArgs(args []string) (taskName, project string, autoConfirm bool) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if arg == "--yes" {
			autoConfirm = true
		} else if taskName == "" {
			taskName = arg
		}
	}
	return taskName, project, autoConfirm
}

var artifactListUsage = "Usage: ctx artifact list <task> [--project=<name>] [--type=proposal|request|response|decision|all]"
var artifactShowUsage = "       ctx artifact show <task> <filename> [--project=<name>]"

func parseArtifactListArgs(args []string) (taskName, project, captureType string) {
	captureType = "all"
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if val, ok := parseFlag(arg, "--type="); ok {
			captureType = val
		} else if taskName == "" {
			taskName = arg
		}
	}
	return taskName, project, captureType
}

var resolveUsage = "Usage: ctx resolve <task> <filename> [--project=<name>]"

func parseArtifactFileArgs(args []string) (taskName, project, filename string) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if taskName == "" {
			taskName = arg
		} else if filename == "" {
			filename = arg
		}
	}
	return taskName, project, filename
}

var taskUsage = []string{"Usage: ctx task <command> [options]", "Commands: start, close, list, show"}

func parseTaskCommandArgs(args []string) (project, taskName string, autoConfirm bool) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if arg == "--yes" {
			autoConfirm = true
		} else if taskName == "" {
			taskName = arg
		}
	}
	return project, taskName, autoConfirm
}

var snapshotUsage = "Usage: ctx snapshot <task> [--project=<name>] [--keep=<n>] [--yes]"

func parseSnapshotArgs(args []string) (project, taskName string, autoConfirm bool, keep int, err error) {
	keep = 10
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--project="):
			project = arg[len("--project="):]
		case arg == "--yes":
			autoConfirm = true
		case strings.HasPrefix(arg, "--keep="):
			val, convErr := strconv.Atoi(arg[len("--keep="):])
			if convErr != nil {
				return "", "", false, 0, fmt.Errorf("invalid --keep value: %s", arg)
			}
			keep = val
		case taskName == "":
			taskName = arg
		}
	}
	if keep <= 0 {
		keep = 10
	}
	return project, taskName, autoConfirm, keep, nil
}
