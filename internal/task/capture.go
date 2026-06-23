package task

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"continuum/internal/filestore"
	"continuum/internal/setup"
)

type CaptureData struct {
	Objective   string
	State       string
	Decisions   string
	Next        string
	Constraints string
	Issues      string
}

type CaptureOptions struct {
	Type        filestore.CaptureType
	AutoConfirm bool
	Resolves    string
}

func isStdinPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func parseCaptureFromMarkdown(content string) *CaptureData {
	sections := parseSections(content)
	decisions := sections["Decisions (Locked)"]
	if strings.TrimSpace(decisions) == "" {
		decisions = sections["Decisions"]
	}
	return &CaptureData{
		Objective:   cleanPrefill(sections["Objective"]),
		State:       cleanPrefill(sections["Current State"]),
		Decisions:   cleanPrefill(decisions),
		Next:        cleanPrefill(sections["Next Step"]),
		Constraints: cleanPrefill(sections["Constraints"]),
		Issues:      cleanPrefill(sections["Active Issues"]),
	}
}

func LoadCaptureData(task, project string) (*CaptureData, error) {
	if err := setup.ValidateTaskName(task); err != nil {
		return nil, err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return nil, err
	}
	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	path, _, err := filestore.LatestSnapshot(taskDir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve snapshot: %w", err)
	}
	if path == "" {
		return &CaptureData{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read snapshot: %w", err)
	}
	return parseCaptureFromMarkdown(string(data)), nil
}

func BuildCaptureSummary(d *CaptureData) string {
	var lines []string
	if d.Objective != "" {
		lines = append(lines, fmt.Sprintf("Objective: %s", d.Objective))
	}
	if d.State != "" {
		lines = append(lines, fmt.Sprintf("State: %s", d.State))
	}
	if d.Decisions != "" {
		lines = append(lines, fmt.Sprintf("Decisions: %s", d.Decisions))
	}
	if d.Next != "" {
		lines = append(lines, fmt.Sprintf("Next: %s", d.Next))
	}
	if d.Constraints != "" {
		lines = append(lines, fmt.Sprintf("Constraints: %s", d.Constraints))
	}
	if d.Issues != "" {
		lines = append(lines, fmt.Sprintf("Issues: %s", d.Issues))
	}
	if len(lines) == 0 {
		return "(no content)"
	}
	return strings.Join(lines, "\n")
}

func saveSnapshot(task, project string, d *CaptureData) (string, error) {
	if err := setup.ValidateTaskName(task); err != nil {
		return "", err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return "", err
	}
	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	name := filestore.NewSnapshotName()
	path := filepath.Join(taskDir, name)

	now := time.Now().Format("2006-01-02 15:04:05 MST")

	content := fmt.Sprintf(`# TASK SNAPSHOT

## Task
%s

## Project
%s

## Objective
%s

## Current State
- %s

## Decisions (Locked)
- %s

## Next Step
- %s

## Constraints
- %s

## Active Issues
- %s

## Last Updated
%s
`,
		task, project,
		d.Objective,
		d.State,
		d.Decisions,
		d.Next,
		d.Constraints,
		d.Issues,
		now,
	)

	if err := filestore.AtomicWrite(path, []byte(content)); err != nil {
		return "", err
	}
	return name, nil
}

func saveRawCapture(task, project string, captureType filestore.CaptureType, raw string) (string, error) {
	if err := setup.ValidateTaskName(task); err != nil {
		return "", err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return "", err
	}
	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	name := filestore.NewCaptureName(captureType)
	path := filepath.Join(taskDir, name)
	now := time.Now().Format("2006-01-02 15:04:05 MST")
	title := strings.ToUpper(string(captureType))
	body := strings.TrimSpace(raw)
	if body != "" {
		body += "\n"
	}

	content := fmt.Sprintf(`# TASK %s

## Task
%s

## Project
%s

## Capture Type
%s

%s
## Last Updated
%s
`, title, task, project, captureType, body, now)

	if err := filestore.AtomicWrite(path, []byte(content)); err != nil {
		return "", err
	}
	return name, nil
}

func appendResolution(raw, resolves string) string {
	raw = strings.TrimSpace(raw)
	if strings.TrimSpace(resolves) == "" {
		if raw == "" {
			return ""
		}
		return raw + "\n"
	}
	var b strings.Builder
	if raw != "" {
		b.WriteString(raw)
		b.WriteString("\n\n")
	}
	b.WriteString("## Resolves\n")
	b.WriteString("- ")
	b.WriteString(resolves)
	b.WriteString("\n")
	return b.String()
}

func buildRawCaptureSummary(captureType filestore.CaptureType, raw string) string {
	firstLine := ""
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			firstLine = strings.TrimPrefix(line, "#")
			firstLine = strings.TrimSpace(firstLine)
			break
		}
	}
	if firstLine == "" {
		firstLine = "(content provided)"
	}
	return fmt.Sprintf("%s: %s", captureType, firstLine)
}

func Capture(task, project string, captureType filestore.CaptureType, autoConfirm bool) error {
	return CaptureWithOptions(task, project, CaptureOptions{
		Type:        captureType,
		AutoConfirm: autoConfirm,
	})
}

func CaptureWithOptions(task, project string, opts CaptureOptions) error {
	if err := setup.ValidateTaskName(task); err != nil {
		return err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return err
	}
	captureType := opts.Type
	if captureType == "" {
		captureType = filestore.StateCapture
	}
	resolves := strings.TrimSpace(opts.Resolves)
	if resolves != "" {
		if captureType != filestore.DecisionCapture {
			return fmt.Errorf("--resolves can only be used with --type=decision")
		}
		if err := validateResolvableArtifactName(resolves); err != nil {
			return err
		}
		if _, err := ReadArtifact(task, project, resolves); err != nil {
			return err
		}
	}

	piped := isStdinPiped()

	if captureType != filestore.StateCapture {
		if !piped {
			return fmt.Errorf("ctx capture --type=%s expects markdown on stdin", captureType)
		}
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
		body := string(raw)
		if resolves != "" {
			body = appendResolution(body, resolves)
		}
		return confirmAndSave(task, buildRawCaptureSummary(captureType, body), opts.AutoConfirm, func() error {
			name, err := saveRawCapture(task, project, captureType, body)
			if err != nil {
				return fmt.Errorf("cannot save %s: %w", captureType, err)
			}
			files := []string{taskFile(project, task, name)}
			if resolves != "" {
				if err := resolveArtifactWithoutCommit(task, project, resolves); err != nil {
					_ = os.Remove(filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task, name))
					return err
				}
				files = append(files, taskFile(project, task, resolves), resolvedArtifactFile(project, task, resolves))
			}
			summary := fmt.Sprintf("%s captured", captureType)
			if err := commitTaskWrite(project, task, string(captureType), summary, files); err != nil {
				return err
			}
			return nil
		}, fmt.Sprintf("%s saved.", captureType))
	}

	var data *CaptureData
	if piped {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
		data = parseCaptureFromMarkdown(string(raw))
	} else {
		var err error
		data, err = LoadCaptureData(task, project)
		if err != nil {
			return err
		}
	}

	if resolves != "" {
		return fmt.Errorf("--resolves can only be used with --type=decision")
	}

	return confirmAndSave(task, BuildCaptureSummary(data), opts.AutoConfirm, func() error {
		name, err := saveSnapshot(task, project, data)
		if err != nil {
			return fmt.Errorf("cannot save snapshot: %w", err)
		}
		if err := commitTaskWrite(project, task, "capture", "snapshot updated", []string{taskFile(project, task, name)}); err != nil {
			return err
		}
		return nil
	}, "State saved.")
}
