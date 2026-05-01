package search

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"continuum/internal/filestore"
	"continuum/internal/setup"
)

type Result struct {
	Project string
	Task    string
	Kind    string
	File    string
	Line    int
	Text    string
}

func Search(query, projectFilter, taskFilter string, limit int, since time.Duration) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if projectFilter != "" {
		if err := setup.ValidateProjectName(projectFilter); err != nil {
			return nil, err
		}
	}
	if taskFilter != "" {
		if err := setup.ValidateTaskName(taskFilter); err != nil {
			return nil, err
		}
	}

	projects, err := resolveProjects(projectFilter)
	if err != nil {
		return nil, err
	}

	var results []Result
	for _, project := range projects {
		projectResults, err := searchProject(project, taskFilter, query, since)
		if err != nil {
			return nil, err
		}
		results = append(results, projectResults...)
	}

	sort.SliceStable(results, func(i, j int) bool {
		leftTS := fileTimestamp(results[i].File)
		rightTS := fileTimestamp(results[j].File)
		if leftTS == rightTS {
			if results[i].File == results[j].File {
				return results[i].Line < results[j].Line
			}
			return results[i].File > results[j].File
		}
		return leftTS > rightTS
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func fileTimestamp(file string) string {
	parts := strings.Split(file, ".")
	if len(parts) >= 4 {
		return parts[1]
	}
	return file
}

func resolveProjects(projectFilter string) ([]string, error) {
	if projectFilter != "" {
		return []string{projectFilter}, nil
	}
	projects, err := setup.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("cannot list projects: %w", err)
	}
	return projects, nil
}

func searchProject(project, taskFilter, query string, since time.Duration) ([]Result, error) {
	tasksDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot read tasks directory for project %q: %w", project, err)
	}

	var taskNames []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if taskFilter != "" && name != taskFilter {
			continue
		}
		taskNames = append(taskNames, name)
	}
	sort.Strings(taskNames)

	var results []Result
	for _, taskName := range taskNames {
		taskDir := filepath.Join(tasksDir, taskName)
		taskResults, err := searchTask(project, taskName, taskDir, query, since)
		if err != nil {
			return nil, err
		}
		results = append(results, taskResults...)
	}
	return results, nil
}

func searchTask(project, taskName, taskDir, query string, since time.Duration) ([]Result, error) {
	var paths []string

	snapshots, err := filestore.AllSnapshots(taskDir)
	if err != nil {
		return nil, fmt.Errorf("cannot list snapshots for %s/%s: %w", project, taskName, err)
	}
	handoffs, err := filestore.AllHandoffs(taskDir)
	if err != nil {
		return nil, fmt.Errorf("cannot list handoffs for %s/%s: %w", project, taskName, err)
	}
	paths = append(paths, snapshots...)
	paths = append(paths, handoffs...)
	for _, captureType := range []filestore.CaptureType{
		filestore.ProposalCapture,
		filestore.RequestCapture,
		filestore.ResponseCapture,
		filestore.DecisionCapture,
	} {
		artifacts, err := filestore.AllCapturesOfType(taskDir, captureType)
		if err != nil {
			return nil, fmt.Errorf("cannot list %s artifacts for %s/%s: %w", captureType, project, taskName, err)
		}
		paths = append(paths, artifacts...)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))

	var results []Result
	for _, path := range paths {
		if since > 0 && !isWithinSince(filepath.Base(path), since) {
			continue
		}
		fileResults, err := searchFile(project, taskName, path, query)
		if err != nil {
			return nil, err
		}
		results = append(results, fileResults...)
	}

	return results, nil
}

func isWithinSince(file string, since time.Duration) bool {
	ts, ok := parseFileTimestamp(file)
	if !ok {
		return false
	}
	return time.Since(ts) <= since
}

func parseFileTimestamp(file string) (time.Time, bool) {
	ts := fileTimestamp(file)
	parsed, err := time.Parse("20060102T150405Z", ts)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func searchFile(project, taskName, path, query string) ([]Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	defer f.Close()

	needle := strings.ToLower(query)
	scanner := bufio.NewScanner(f)
	lineNo := 0
	var results []Result

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if !strings.Contains(strings.ToLower(line), needle) {
			continue
		}
		results = append(results, Result{
			Project: project,
			Task:    taskName,
			Kind:    fileKind(path),
			File:    filepath.Base(path),
			Line:    lineNo,
			Text:    strings.TrimSpace(line),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("cannot scan %s: %w", path, err)
	}

	return results, nil
}

func fileKind(path string) string {
	base := filepath.Base(path)
	switch {
	case strings.HasPrefix(base, "snapshot."):
		return "snapshot"
	case strings.HasPrefix(base, "handoff."):
		return "handoff"
	case strings.HasPrefix(base, "proposal."):
		return "proposal"
	case strings.HasPrefix(base, "request."):
		return "request"
	case strings.HasPrefix(base, "response."):
		return "response"
	case strings.HasPrefix(base, "decision."):
		return "decision"
	default:
		return "file"
	}
}
