package task

import (
	"fmt"
	"path/filepath"

	"continuum/internal/events"
	"continuum/internal/identity"
	"continuum/internal/setup"
)

func commitTaskWrite(project, task, operation, summary string, files []string) error {
	detail := summary
	if detail == "" {
		detail = operation
	}
	eventFile := ""
	if len(files) > 0 {
		eventFile = files[0]
	}
	if err := events.AppendWithFile(project, task, operation, "ok", detail, eventFile); err == nil {
		files = append([]string{events.ActivityRelPath()}, files...)
	}

	message := buildCommitMessage(project, task, operation, summary)
	if err := setup.CommitFiles(message, files); err != nil {
		return fmt.Errorf("cannot save git history: %w", err)
	}
	setup.PushBestEffort()
	return nil
}

func taskFile(project, task, name string) string {
	return filepath.ToSlash(filepath.Join("projects", project, "tasks", task, name))
}

func buildCommitMessage(project, task, operation, summary string) string {
	hostname := identity.HostName()
	agent := identity.AgentName()
	target := project
	if task != "" {
		target = fmt.Sprintf("%s/%s", project, task)
	}

	return fmt.Sprintf(
		"%s(%s): %s\n\ncontinuum/%s\nmachine: %s\nagent: %s",
		operation,
		target,
		summary,
		target,
		hostname,
		agent,
	)
}
