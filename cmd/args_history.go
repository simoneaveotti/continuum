package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var historyUsage = "Usage: ctx history [--project=<name>] [--task=<name>] [--limit=<n>] [--since=<duration>]"

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
			return "", "", 0, 0, errors.New(historyUsage)
		}
	}
	return project, taskName, limit, since, nil
}

var diffUsage = "Usage: ctx diff <task> [<from-snapshot> <to-snapshot>] [--project=<name>]"

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
			return "", "", "", "", errors.New(diffUsage)
		}
	}
	if taskName == "" {
		return "", "", "", "", errors.New(diffUsage)
	}
	if (fromName == "" && toName != "") || (fromName != "" && toName == "") {
		return "", "", "", "", fmt.Errorf("provide both snapshot names or neither")
	}
	return project, taskName, fromName, toName, nil
}

var searchUsage = "Usage: ctx search [--project=<name>] [--task=<name>] [--limit=<n>] [--since=<duration>] <query>"

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
		return "", "", "", 0, 0, errors.New(searchUsage)
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
