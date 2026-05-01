package task

import (
	"fmt"
	"os"
	"path/filepath"

	"continuum/internal/events"
	"continuum/internal/filestore"
	"continuum/internal/setup"
)

func SnapshotClean(task, project string, keep int) (int, error) {
	if err := setup.ValidateTaskName(task); err != nil {
		return 0, err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return 0, err
	}
	if keep <= 0 {
		keep = 10
	}

	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	snapshots, err := filestore.AllSnapshots(taskDir)
	if err != nil {
		return 0, fmt.Errorf("cannot list snapshots: %w", err)
	}
	if len(snapshots) <= keep {
		return 0, nil
	}
	removeCount := len(snapshots) - keep
	toRemove := snapshots[:removeCount]

	for _, path := range toRemove {
		if err := os.Remove(path); err != nil {
			return 0, fmt.Errorf("cannot remove snapshot %s: %w", path, err)
		}
	}

	relDir := filepath.ToSlash(filepath.Join("projects", project, "tasks", task))
	if err := setup.StageDeletedPaths([]string{relDir}); err != nil {
		return 0, err
	}

	if err := commitTaskWrite(project, task, "clean", fmt.Sprintf("retained last %d snapshots", keep), nil); err != nil {
		return 0, err
	}

	return removeCount, nil
}

func DeleteTask(task, project string) error {
	if err := setup.ValidateTaskName(task); err != nil {
		return err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return err
	}

	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	if _, err := os.Stat(taskDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("task '%s' not found", task)
		}
		return fmt.Errorf("cannot stat task directory: %w", err)
	}

	if err := os.RemoveAll(taskDir); err != nil {
		return fmt.Errorf("cannot remove task directory: %w", err)
	}

	taskParent := filepath.ToSlash(filepath.Join("projects", project, "tasks"))
	if err := setup.StageDeletedPaths([]string{taskParent}); err != nil {
		return err
	}

	if err := events.Append(project, task, "task_deleted", "ok", "task removed"); err == nil {
		_ = setup.StagePaths([]string{events.ActivityRelPath()})
	}

	return commitTaskWrite(project, task, "delete", "task removed", nil)
}
