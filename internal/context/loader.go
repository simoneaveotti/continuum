package context

import (
	"fmt"
	"os"
	"path/filepath"

	"continuum/internal/filestore"
	"continuum/internal/setup"
)

type ContextData struct {
	Profile      string
	Project      string
	Snapshot     string
	SnapshotName string // filename of the snapshot used, for source traceability
	Handoff      string
	Unsynced     []string
	TaskContexts map[string]*ContextData
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", path, err)
	}
	return string(data), nil
}

func Load(task, project string) (*ContextData, error) {
	return load(task, project, true)
}

func load(task, project string, pull bool) (*ContextData, error) {
	if err := setup.ValidateTaskName(task); err != nil {
		return nil, err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return nil, err
	}

	base := setup.ContinuumPath()

	if pull {
		if err := setup.PullLatest(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
	}

	profile, err := readFile(filepath.Join(base, "profile.md"))
	if err != nil {
		return nil, err
	}

	projectData, err := readFile(filepath.Join(base, "projects", project, "project.md"))
	if err != nil {
		return nil, err
	}

	taskDir := filepath.Join(base, "projects", project, "tasks", task)
	if info, err := os.Stat(taskDir); err != nil || !info.IsDir() {
		if os.IsNotExist(err) || err == nil {
			return nil, fmt.Errorf("task %q not found in project %q", task, project)
		}
		return nil, fmt.Errorf("cannot access task %q in project %q: %w", task, project, err)
	}

	var snapshot, snapshotName string
	if snapshotPath, name, err := filestore.LatestSnapshot(taskDir); err == nil && snapshotPath != "" {
		if data, err := readFile(snapshotPath); err == nil {
			snapshot = data
			snapshotName = name
		}
	}

	var handoff string
	if handoffPath, _, err := filestore.LatestHandoff(taskDir); err == nil && handoffPath != "" {
		if data, err := readFile(handoffPath); err == nil {
			handoff = data
		}
	}

	return &ContextData{
		Profile:      profile,
		Project:      projectData,
		Snapshot:     snapshot,
		SnapshotName: snapshotName,
		Handoff:      handoff,
		Unsynced:     setup.UnsyncedCommits(),
	}, nil
}
