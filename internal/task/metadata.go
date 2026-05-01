package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"continuum/internal/setup"
)

const metadataFileName = "task.json"

type Status string

const (
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

type TaskInfo struct {
	Name   string
	Status Status
}

type taskMetadata struct {
	Status Status `json:"status"`
}

func (s Status) Valid() bool {
	return s == StatusActive || s == StatusClosed
}

func parseStatus(value string) (Status, error) {
	status := Status(value)
	if !status.Valid() {
		return "", fmt.Errorf("invalid task status %q", value)
	}
	return status, nil
}

func readStatus(taskDir string) (Status, error) {
	path := filepath.Join(taskDir, metadataFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return StatusActive, nil
		}
		return "", fmt.Errorf("cannot read task metadata: %w", err)
	}

	var meta taskMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", fmt.Errorf("cannot parse task metadata: %w", err)
	}
	if meta.Status == "" {
		return StatusActive, nil
	}
	if !meta.Status.Valid() {
		return "", fmt.Errorf("invalid task metadata status %q", meta.Status)
	}
	return meta.Status, nil
}

func writeStatus(taskDir string, status Status) error {
	if !status.Valid() {
		return fmt.Errorf("invalid task status %q", status)
	}
	data, err := json.MarshalIndent(taskMetadata{Status: status}, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot encode task metadata: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(taskDir, metadataFileName), data, 0o644); err != nil {
		return fmt.Errorf("cannot write task metadata: %w", err)
	}
	return nil
}

func SetStatus(task, project string, status Status) (bool, error) {
	if err := setup.ValidateTaskName(task); err != nil {
		return false, err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return false, err
	}
	if !status.Valid() {
		return false, fmt.Errorf("invalid task status %q", status)
	}

	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	if info, err := os.Stat(taskDir); err != nil || !info.IsDir() {
		if os.IsNotExist(err) || err == nil {
			return false, fmt.Errorf("task '%s' not found", task)
		}
		return false, fmt.Errorf("cannot stat task directory: %w", err)
	}

	current, err := readStatus(taskDir)
	if err != nil {
		return false, err
	}
	if current == status {
		return false, nil
	}

	if err := writeStatus(taskDir, status); err != nil {
		return false, err
	}

	operation := "task_reopened"
	summary := "task reopened"
	if status == StatusClosed {
		operation = "task_closed"
		summary = "task closed"
	}

	if err := commitTaskWrite(project, task, operation, summary, []string{taskFile(project, task, metadataFileName)}); err != nil {
		return false, err
	}
	return true, nil
}
