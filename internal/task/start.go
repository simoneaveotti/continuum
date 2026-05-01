package task

import (
	"fmt"
	"os"
	"path/filepath"

	"continuum/internal/events"
	"continuum/internal/setup"
)

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func ensureFile(path, content string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

const notesTemplate = `# TASK NOTES

- ...
`

type StartResult int

const (
	StartCreated StartResult = iota
	StartAlreadyActive
)

func Start(task, project string) (StartResult, error) {
	if project == "" {
		project = "default"
	}
	if err := setup.ValidateTaskName(task); err != nil {
		return StartCreated, err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return StartCreated, err
	}

	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	if info, err := os.Stat(taskDir); err == nil && info.IsDir() {
		status, err := readStatus(taskDir)
		if err != nil {
			return StartCreated, err
		}
		if status == StatusActive {
			return StartAlreadyActive, nil
		}
		return StartCreated, fmt.Errorf("task '%s' already exists and is closed; use `ctx task reopen %s --project=%s`", task, task, project)
	} else if err != nil && !os.IsNotExist(err) {
		return StartCreated, fmt.Errorf("cannot access task directory %s: %w", taskDir, err)
	}

	if err := ensureDir(taskDir); err != nil {
		return StartCreated, fmt.Errorf("cannot create task directory %s: %w", taskDir, err)
	}

	if err := ensureFile(filepath.Join(taskDir, "notes.md"), notesTemplate); err != nil {
		return StartCreated, fmt.Errorf("cannot create notes.md: %w", err)
	}
	if _, err := os.Stat(filepath.Join(taskDir, metadataFileName)); os.IsNotExist(err) {
		if err := writeStatus(taskDir, StatusActive); err != nil {
			return StartCreated, fmt.Errorf("cannot create %s: %w", metadataFileName, err)
		}
	} else if err != nil {
		return StartCreated, fmt.Errorf("cannot access %s: %w", metadataFileName, err)
	}

	notesFile := filepath.ToSlash(filepath.Join("projects", project, "tasks", task, "notes.md"))
	metadataFile := filepath.ToSlash(filepath.Join("projects", project, "tasks", task, metadataFileName))
	files := []string{notesFile, metadataFile}
	if err := events.Append(project, task, "task_started", "ok", "task created"); err == nil {
		files = append([]string{events.ActivityRelPath()}, files...)
	}
	if err := setup.CommitFiles(buildCommitMessage(project, task, "start", "task initialized"), files); err != nil {
		return StartCreated, fmt.Errorf("cannot save git history: %w", err)
	}
	setup.PushBestEffort()

	fmt.Printf("Task '%s' initialized in .ctx/projects/%s/tasks/%s\n", task, project, task)
	return StartCreated, nil
}
