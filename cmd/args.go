package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

func parseSyncArgs(args []string) (remote, prefer string, force bool, err error) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--remote="); ok {
			remote = val
		} else if val, ok := parseFlag(arg, "--prefer="); ok {
			prefer = val
		} else if arg == "--force" {
			force = true
		} else {
			return "", "", false, fmt.Errorf("Usage: ctx sync [--remote=<url>] [--prefer=local|remote] [--force]")
		}
	}
	if prefer != "" && prefer != "local" && prefer != "remote" {
		return "", "", false, fmt.Errorf("invalid --prefer value: %q (expected local or remote)", prefer)
	}
	if force && prefer == "" {
		return "", "", false, fmt.Errorf("Usage: ctx sync [--remote=<url>] [--prefer=local|remote] [--force]")
	}
	return remote, prefer, force, nil
}

func parseWatchArgs(args []string) (project string, interval time.Duration, tui bool, err error) {
	interval = 2 * time.Second
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if arg == "--tui" {
			tui = true
		} else if val, ok := parseFlag(arg, "--interval="); ok {
			parsed, parseErr := time.ParseDuration(val)
			if parseErr != nil {
				return "", 0, false, fmt.Errorf("invalid interval: %w", parseErr)
			}
			interval = parsed
		} else {
			return "", 0, false, fmt.Errorf("Usage: ctx watch [--project=<name>] [--interval=<duration>] [--tui]")
		}
	}
	return project, interval, tui, nil
}

func parseSearchArgs(args []string) (project, taskName, query string, err error) {
	project, taskName, query, _, _, err = parseSearchArgsFull(args)
	return project, taskName, query, err
}

func parseHistoryArgs(args []string) (project, taskName string, limit int, since time.Duration, err error) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if val, ok := parseFlag(arg, "--task="); ok {
			taskName = val
		} else if val, ok := parseFlag(arg, "--limit="); ok {
			parsed, parseErr := strconv.Atoi(val)
			if parseErr != nil || parsed <= 0 {
				return "", "", 0, 0, fmt.Errorf("invalid --limit value: %q", val)
			}
			limit = parsed
		} else if val, ok := parseFlag(arg, "--since="); ok {
			parsed, parseErr := parseSinceValue(val)
			if parseErr != nil {
				return "", "", 0, 0, parseErr
			}
			since = parsed
		} else {
			return "", "", 0, 0, fmt.Errorf("Usage: ctx history [--project=<name>] [--task=<name>] [--limit=<n>] [--since=<duration>]")
		}
	}
	return project, taskName, limit, since, nil
}

func parseDiffArgs(args []string) (project, taskName, fromName, toName string, err error) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if taskName == "" {
			taskName = arg
		} else if fromName == "" {
			fromName = arg
		} else if toName == "" {
			toName = arg
		} else {
			return "", "", "", "", fmt.Errorf("Usage: ctx diff <task> [<from-snapshot> <to-snapshot>] [--project=<name>]")
		}
	}
	if taskName == "" {
		return "", "", "", "", fmt.Errorf("Usage: ctx diff <task> [<from-snapshot> <to-snapshot>] [--project=<name>]")
	}
	if (fromName == "" && toName != "") || (fromName != "" && toName == "") {
		return "", "", "", "", fmt.Errorf("provide both snapshot names or neither")
	}
	return project, taskName, fromName, toName, nil
}

func parseSearchArgsWithLimit(args []string) (project, taskName, query string, limit int, err error) {
	project, taskName, query, limit, _, err = parseSearchArgsFull(args)
	return project, taskName, query, limit, err
}

func parseSearchArgsFull(args []string) (project, taskName, query string, limit int, since time.Duration, err error) {
	var queryParts []string
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if val, ok := parseFlag(arg, "--task="); ok {
			taskName = val
		} else if val, ok := parseFlag(arg, "--limit="); ok {
			parsed, parseErr := strconv.Atoi(val)
			if parseErr != nil || parsed <= 0 {
				return "", "", "", 0, 0, fmt.Errorf("invalid --limit value: %q", val)
			}
			limit = parsed
		} else if val, ok := parseFlag(arg, "--since="); ok {
			parsed, parseErr := parseSinceValue(val)
			if parseErr != nil {
				return "", "", "", 0, 0, parseErr
			}
			since = parsed
		} else {
			queryParts = append(queryParts, arg)
		}
	}
	query = strings.TrimSpace(strings.Join(queryParts, " "))
	if query == "" {
		return "", "", "", 0, 0, fmt.Errorf("Usage: ctx search [--project=<name>] [--task=<name>] [--limit=<n>] [--since=<duration>] <query>")
	}
	return project, taskName, query, limit, since, nil
}

func parseSinceValue(val string) (time.Duration, error) {
	if strings.HasSuffix(val, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(val, "d"))
		if err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid --since value: %q", val)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	parsed, err := time.ParseDuration(val)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid --since value: %q", val)
	}
	return parsed, nil
}

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
		return "", false, fmt.Errorf("Usage: ctx agent status [--project=<name>]\n       ctx agent update [--project=<name>] [--force]")
	}
	return projectName, force, nil
}

func parseConfigSetArgs(args []string) (key, value string, err error) {
	if len(args) != 2 {
		return "", "", fmt.Errorf("Usage: ctx config set host <name>")
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
			return nil, "", "", "", false, fmt.Errorf("Usage: ctx export [<task> | --project=<name[,name2...]> | --session] [--path=<destination>] [--encrypt[=<algo>]]")
		}
	}
	if session && (taskName != "" || len(projects) > 0) {
		return nil, "", "", "", false, fmt.Errorf("cannot combine --session with task or --project")
	}
	if taskName != "" && len(projects) > 1 {
		return nil, "", "", "", false, fmt.Errorf("task export accepts at most one project")
	}
	if taskName == "" && !session && len(projects) == 0 {
		return nil, "", "", "", false, fmt.Errorf("Usage: ctx export [<task> | --project=<name[,name2...]> | --session] [--path=<destination>] [--encrypt[=<algo>]]")
	}
	return projects, taskName, customPath, encryptAlgo, session, nil
}

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

func parseListArgs(args []string) (project, status string, err error) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if val, ok := parseFlag(arg, "--status="); ok {
			status = val
		} else if arg == "--all" {
			status = "all"
		} else {
			return "", "", fmt.Errorf("Usage: ctx list [--project=<name>] [--all | --status=<active|closed>]")
		}
	}
	return project, status, nil
}

func parseTaskCommandArgs(args []string) (project, taskName string) {
	for _, arg := range args {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if taskName == "" {
			taskName = arg
		}
	}
	return project, taskName
}

func parseProjectCommandArgs(args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("expected exactly one project name")
	}
	if strings.HasPrefix(args[0], "--") {
		return "", fmt.Errorf("project commands accept a positional project name, not flags")
	}
	return args[0], nil
}

func parseProjectOnboardArgs(args []string) (project string, force bool, autoConfirm bool, err error) {
	for _, arg := range args {
		if arg == "--force" {
			force = true
		} else if arg == "--yes" {
			autoConfirm = true
		} else if strings.HasPrefix(arg, "--") {
			return "", false, false, fmt.Errorf("Usage: ctx project onboard <project> [--force] [--yes]")
		} else if project == "" {
			project = arg
		} else {
			return "", false, false, fmt.Errorf("Usage: ctx project onboard <project> [--force] [--yes]")
		}
	}
	if project == "" {
		return "", false, false, fmt.Errorf("Usage: ctx project onboard <project> [--force] [--yes]")
	}
	return project, force, autoConfirm, nil
}

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
