package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"continuum/internal/setup"
)

func List(project string) ([]string, error) {
	items, err := ListWithStatus(project, string(StatusActive))
	if err != nil {
		return nil, err
	}
	tasks := make([]string, 0, len(items))
	for _, item := range items {
		tasks = append(tasks, item.Name)
	}
	return tasks, nil
}

func ListWithStatus(project, statusFilter string) ([]TaskInfo, error) {
	if err := setup.ValidateProjectName(project); err != nil {
		return nil, err
	}
	if statusFilter == "" {
		statusFilter = string(StatusActive)
	}
	if statusFilter != "all" {
		if _, err := parseStatus(statusFilter); err != nil {
			return nil, err
		}
	}

	tasksDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks")

	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read tasks directory %s: %w", tasksDir, err)
	}

	var tasks []TaskInfo

	for _, entry := range entries {
		if entry.IsDir() {
			taskDir := filepath.Join(tasksDir, entry.Name())
			status, err := readStatus(taskDir)
			if err != nil {
				return nil, err
			}
			if statusFilter != "all" && string(status) != statusFilter {
				continue
			}
			tasks = append(tasks, TaskInfo{Name: entry.Name(), Status: status})
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Name < tasks[j].Name
	})
	return tasks, nil
}
