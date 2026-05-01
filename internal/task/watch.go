package task

import (
	"fmt"
	"strings"
	"time"

	"continuum/internal/events"
)

type watchState struct {
	Offset int64
}

func Watch(project string, interval time.Duration) error {
	if interval <= 0 {
		interval = 2 * time.Second
	}

	scope, err := watchProjects(project)
	if err != nil {
		return err
	}
	if project != "" && len(scope) == 0 {
		return fmt.Errorf("no projects found")
	}

	if project == "" {
		fmt.Printf("Watching all projects every %s. Press Ctrl+C to stop.\n", interval)
	} else {
		fmt.Printf("Watching project '%s' every %s. Press Ctrl+C to stop.\n", project, interval)
	}

	state, err := collectWatchState()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C

		next, err := collectWatchState()
		if err != nil {
			return err
		}

		printWatchChanges(scope, state, next)
		state = next
	}
}

func watchProjects(project string) ([]string, error) {
	if project != "" {
		return []string{project}, nil
	}
	return nil, nil
}

func collectWatchState() (map[string]watchState, error) {
	_, offset, err := events.ReadFromOffset(0)
	if err != nil {
		return nil, err
	}
	return map[string]watchState{"activity": {Offset: offset}}, nil
}

func watchKey(project, task string) string {
	return project + "/" + task
}

func splitWatchKey(key string) (string, string) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return "", key
	}
	return parts[0], parts[1]
}

func printWatchChanges(projects []string, previous, next map[string]watchState) {
	before := previous["activity"].Offset
	after := next["activity"].Offset
	if after <= before {
		return
	}
	items, _, err := events.ReadFromOffset(before)
	if err != nil {
		fmt.Println("watch error:", err)
		return
	}
	allowed := make(map[string]struct{}, len(projects))
	for _, project := range projects {
		allowed[project] = struct{}{}
	}
	for _, item := range items {
		if item.Project != "" && len(allowed) > 0 {
			if _, ok := allowed[item.Project]; !ok {
				continue
			}
		}
		printWatchEvent(item)
	}
}

func printWatchEvent(item events.Event) {
	target := eventTarget(item)
	if target == "" {
		target = "-"
	}
	detail := item.Detail
	if detail == "" {
		detail = item.Status
	}
	fmt.Printf("[%s] %s %s %s@%s %s\n", item.Timestamp, target, item.Type, item.Agent, item.Host, detail)
}

func eventTarget(item events.Event) string {
	target := item.Project
	if item.Task != "" {
		target = item.Project + "/" + item.Task
	}
	return target
}
