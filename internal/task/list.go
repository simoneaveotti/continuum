package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func List() ([]string, error) {
	tasksDir := filepath.Join(".continuum", "tasks")

	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read tasks directory %s: %w", tasksDir, err)
	}

	var tasks []string

	for _, entry := range entries {
		if entry.IsDir() {
			tasks = append(tasks, entry.Name())
		}
	}

	sort.Strings(tasks)
	return tasks, nil
}
