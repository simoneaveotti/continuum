package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"continuum/internal/filestore"
	"continuum/internal/setup"
)

func Diff(task, project, fromName, toName string) (string, error) {
	if err := setup.ValidateTaskName(task); err != nil {
		return "", err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return "", err
	}

	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	snapshots, err := filestore.AllSnapshots(taskDir)
	if err != nil {
		return "", fmt.Errorf("cannot list snapshots: %w", err)
	}
	if len(snapshots) == 0 {
		return "", fmt.Errorf("no snapshots found for task %q", task)
	}

	fromPath, toPath, err := resolveDiffTargets(task, snapshots, fromName, toName)
	if err != nil {
		return "", err
	}

	fromData, err := os.ReadFile(fromPath)
	if err != nil {
		return "", fmt.Errorf("cannot read snapshot %q: %w", filepath.Base(fromPath), err)
	}
	toData, err := os.ReadFile(toPath)
	if err != nil {
		return "", fmt.Errorf("cannot read snapshot %q: %w", filepath.Base(toPath), err)
	}

	diff := unifiedDiff(filepath.Base(fromPath), string(fromData), filepath.Base(toPath), string(toData))
	if strings.TrimSpace(diff) == "" {
		return "No differences.\n", nil
	}
	return diff, nil
}

func resolveDiffTargets(task string, snapshots []string, fromName, toName string) (string, string, error) {
	switch {
	case fromName == "" && toName == "":
		if len(snapshots) < 2 {
			return "", "", fmt.Errorf("task %q needs at least two snapshots for diff", task)
		}
		return snapshots[len(snapshots)-2], snapshots[len(snapshots)-1], nil
	case fromName == "" || toName == "":
		return "", "", fmt.Errorf("provide both snapshot names or neither")
	default:
		fromPath := findSnapshotByName(snapshots, fromName)
		if fromPath == "" {
			return "", "", fmt.Errorf("snapshot %q not found for task %q", fromName, task)
		}
		toPath := findSnapshotByName(snapshots, toName)
		if toPath == "" {
			return "", "", fmt.Errorf("snapshot %q not found for task %q", toName, task)
		}
		return fromPath, toPath, nil
	}
}

func findSnapshotByName(paths []string, name string) string {
	for _, path := range paths {
		if filepath.Base(path) == name {
			return path
		}
	}
	return ""
}

func unifiedDiff(fromName, from, toName, to string) string {
	a := splitLines(from)
	b := splitLines(to)

	type step struct {
		op   byte
		line string
	}

	dp := make([][]int, len(a)+1)
	for i := range dp {
		dp[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var steps []step
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			steps = append(steps, step{op: ' ', line: a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			steps = append(steps, step{op: '-', line: a[i]})
			i++
		default:
			steps = append(steps, step{op: '+', line: b[j]})
			j++
		}
	}
	for ; i < len(a); i++ {
		steps = append(steps, step{op: '-', line: a[i]})
	}
	for ; j < len(b); j++ {
		steps = append(steps, step{op: '+', line: b[j]})
	}

	changed := false
	var bld strings.Builder
	bld.WriteString("--- " + fromName + "\n")
	bld.WriteString("+++ " + toName + "\n")
	for _, s := range steps {
		if s.op != ' ' {
			changed = true
		}
		bld.WriteByte(s.op)
		bld.WriteString(s.line)
		bld.WriteByte('\n')
	}
	if !changed {
		return ""
	}
	return bld.String()
}

func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
