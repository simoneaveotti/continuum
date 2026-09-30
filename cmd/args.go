package main

import (
	"errors"
	"fmt"
	"strings"

	"continuum/internal/export"
)

// parseFlag extracts the value from a flag like "--name=value".
// Returns ("", false) if the arg doesn't match the prefix or has no value.
func parseFlag(arg, prefix string) (string, bool) {
	if !strings.HasPrefix(arg, prefix) {
		return "", false
	}
	val := arg[len(prefix):]
	return val, true
}

var initUsage = "Usage: ctx init [--templates=<path>] [--remote=<url>] [--force]"

func parseInitArgs(args []string) (projectName, templatesPath string, force bool, remote string) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--templates="); ok {
			templatesPath = val
		} else if val, ok := parseFlag(arg, "--remote="); ok {
			remote = val
		} else if arg == "--force" {
			force = true
		} else {
			return arg, templatesPath, force, remote
		}
	}
	return projectName, templatesPath, force, remote
}

func parseContextArgs(args []string) (project, taskName string, compact bool) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if arg == "--compact" {
			compact = true
		} else if taskName == "" {
			taskName = arg
		}
	}
	return project, taskName, compact
}

var syncUsage = "Usage: ctx sync [--remote=<url>] [--prefer=local|remote] [--force]"

func parseSyncArgs(args []string) (remote, prefer string, force bool, err error) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--remote="); ok {
			remote = val
		} else if val, ok := parseFlag(arg, "--prefer="); ok {
			prefer = val
		} else if arg == "--force" {
			force = true
		} else {
			return "", "", false, errors.New(syncUsage)
		}
	}
	if prefer != "" && prefer != "local" && prefer != "remote" {
		return "", "", false, fmt.Errorf("invalid --prefer value: %q (expected local or remote)", prefer)
	}
	if force && prefer == "" {
		return "", "", false, errors.New(syncUsage)
	}
	return remote, prefer, force, nil
}

var configSetUsage = "Usage: ctx config set host <name>"

func parseConfigSetArgs(args []string) (key, value string, err error) {
	if len(args) != 2 {
		return "", "", errors.New(configSetUsage)
	}
	return args[0], args[1], nil
}

func parseProjectsValue(value string) []string {
	var items []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

var exportUsage = "Usage: ctx export [<task> | --project=<name[,name2...]> | --session] [--path=<destination>] [--encrypt[=<algo>]]"

func parseExportArgs(args []string) (projects []string, taskName, customPath string, encryptAlgo export.EncryptionAlgo, session bool, err error) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			projects = parseProjectsValue(val)
		} else if val, ok := parseFlag(arg, "--path="); ok {
			customPath = val
		} else if val, ok := parseFlag(arg, "--encrypt="); ok {
			encryptAlgo = export.EncryptionAlgo(val)
		} else if arg == "--encrypt" {
			encryptAlgo = export.AlgoAES_GCM_V2
		} else if arg == "--session" {
			session = true
		} else if taskName == "" {
			taskName = arg
		} else {
			return nil, "", "", "", false, errors.New(exportUsage)
		}
	}
	if session && (taskName != "" || len(projects) > 0) {
		return nil, "", "", "", false, fmt.Errorf("cannot combine --session with task or --project")
	}
	if taskName != "" && len(projects) > 1 {
		return nil, "", "", "", false, fmt.Errorf("task export accepts at most one project")
	}
	if taskName == "" && !session && len(projects) == 0 {
		return nil, "", "", "", false, errors.New(exportUsage)
	}
	return projects, taskName, customPath, encryptAlgo, session, nil
}

var importUsage = "Usage: ctx import <path> [--decrypt[=<algo>]]"

func parseImportArgs(args []string) (zipPath string, decrypt bool, algo export.EncryptionAlgo) {
	if len(args) > 0 {
		zipPath = args[0]
	}
	for _, arg := range args[1:] {
		if val, ok := parseFlag(arg, "--decrypt="); ok {
			decrypt = true
			algo = export.EncryptionAlgo(val)
		} else if arg == "--decrypt" {
			decrypt = true
		}
	}
	return zipPath, decrypt, algo
}

var listUsage = "Usage: ctx list [--project=<name>] [--all | --status=<active|closed>]"

func parseListArgs(args []string) (project, status string, err error) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if val, ok := parseFlag(arg, "--status="); ok {
			status = val
		} else if arg == "--all" {
			status = "all"
		} else {
			return "", "", errors.New(listUsage)
		}
	}
	return project, status, nil
}

var resumeUsage = "Usage: ctx resume [--verbose]"

func parseResumeArgs(args []string) (verbose bool, err error) {
	for _, arg := range args {
		if arg == "--verbose" {
			verbose = true
			continue
		}
		return false, errors.New(resumeUsage)
	}
	return verbose, nil
}

var helpUsage = "Usage: ctx --help [--verbose]"

func parseHelpArgs(args []string) (verbose bool, err error) {
	for _, arg := range args {
		if arg == "--verbose" {
			verbose = true
			continue
		}
		return false, errors.New(helpUsage)
	}
	return verbose, nil
}
