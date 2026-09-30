package main

import (
	"errors"
	"fmt"
	"time"
)

var watchUsage = "Usage: ctx watch [--project=<name>] [--interval=<duration>] [--tui]"

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
			return "", 0, false, errors.New(watchUsage)
		}
	}
	return project, interval, tui, nil
}
