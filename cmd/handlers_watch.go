package main

import "continuum/internal/task"

func handleWatch(args []string) {
	project, interval, tui, err := parseWatchArgs(args)
	if err != nil {
		die(err)
	}
	if tui {
		err = task.WatchTUI(project, interval)
	} else {
		err = task.Watch(project, interval)
	}
	if err != nil {
		die(err)
	}
}
