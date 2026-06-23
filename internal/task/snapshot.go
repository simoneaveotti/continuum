package task

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"continuum/internal/filestore"
	"continuum/internal/setup"
)

type SnapshotData struct {
	Task               string
	Objective          string
	CurrentState       string
	Decisions          string
	RelevantFiles      string
	LastStep           string
	NextStep           string
	ActiveIssues       string
	Constraints        string
	ThingsNotToRevisit string
}

func parseSnapshotFromMarkdown(task string, content string) *SnapshotData {
	sections := parseSections(content)
	return &SnapshotData{
		Task:               task,
		Objective:          cleanPrefill(sections["Objective"]),
		CurrentState:       cleanPrefill(sections["Current State"]),
		Decisions:          cleanPrefill(sections["Decisions (Locked)"]),
		RelevantFiles:      cleanPrefill(sections["Relevant Files"]),
		LastStep:           cleanPrefill(sections["Last Step Completed"]),
		NextStep:           cleanPrefill(sections["Next Step"]),
		ActiveIssues:       cleanPrefill(sections["Active Issues"]),
		Constraints:        cleanPrefill(sections["Constraints"]),
		ThingsNotToRevisit: cleanPrefill(sections["Things Not To Revisit"]),
	}
}

func buildSnapshotSummary(s *SnapshotData) string {
	var lines []string
	if s.Objective != "" {
		lines = append(lines, fmt.Sprintf("Objective: %s", s.Objective))
	}
	if s.CurrentState != "" {
		lines = append(lines, fmt.Sprintf("Current State: %s", s.CurrentState))
	}
	if s.NextStep != "" {
		lines = append(lines, fmt.Sprintf("Next Step: %s", s.NextStep))
	}
	if s.ActiveIssues != "" {
		lines = append(lines, fmt.Sprintf("Active Issues: %s", s.ActiveIssues))
	}
	if len(lines) == 0 {
		return "(no content)"
	}
	return strings.Join(lines, "\n")
}

func buildSnapshotMarkdown(s SnapshotData) string {
	now := time.Now().Format("2006-01-02 15:04:05 MST")

	return fmt.Sprintf(`# TASK SNAPSHOT

## Task
%s

## Objective
%s

## Current State
- %s

## Decisions (Locked)
- %s

## Relevant Files
- %s

## Last Step Completed
- %s

## Next Step
- %s

## Active Issues
- %s

## Constraints
- %s

## Things Not To Revisit
- %s

## Last Updated
%s
`,
		s.Task,
		s.Objective,
		s.CurrentState,
		s.Decisions,
		s.RelevantFiles,
		s.LastStep,
		s.NextStep,
		s.ActiveIssues,
		s.Constraints,
		s.ThingsNotToRevisit,
		now,
	)
}

func loadExistingSnapshot(task, project string) map[string]string {
	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	path, _, err := filestore.LatestSnapshot(taskDir)
	if err != nil || path == "" {
		return map[string]string{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}

	sections := parseSections(string(data))
	for k, v := range sections {
		sections[k] = cleanPrefill(v)
	}
	return sections
}

func SnapshotRefresh(task, project string, autoConfirm bool) error {
	if err := setup.ValidateTaskName(task); err != nil {
		return err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return err
	}

	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	if _, err := os.Stat(taskDir); err != nil {
		return fmt.Errorf("task directory not found: %s", taskDir)
	}

	piped := isStdinPiped()

	var data SnapshotData
	if piped {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
		data = *parseSnapshotFromMarkdown(task, string(raw))
	} else {
		reader := bufio.NewReader(os.Stdin)
		existing := loadExistingSnapshot(task, project)

		fmt.Printf("Updating snapshot for task '%s'\n", task)

		objective, err := promptWithDefault(reader, "Objective", existing["Objective"])
		if err != nil {
			return err
		}
		currentState, err := promptWithDefault(reader, "Current State", existing["Current State"])
		if err != nil {
			return err
		}
		decisions, err := promptWithDefault(reader, "Decisions (Locked)", existing["Decisions (Locked)"])
		if err != nil {
			return err
		}
		relevantFiles, err := promptWithDefault(reader, "Relevant Files", existing["Relevant Files"])
		if err != nil {
			return err
		}
		lastStep, err := promptWithDefault(reader, "Last Step Completed", existing["Last Step Completed"])
		if err != nil {
			return err
		}
		nextStep, err := promptWithDefault(reader, "Next Step", existing["Next Step"])
		if err != nil {
			return err
		}
		activeIssues, err := promptWithDefault(reader, "Active Issues", existing["Active Issues"])
		if err != nil {
			return err
		}
		constraints, err := promptWithDefault(reader, "Constraints", existing["Constraints"])
		if err != nil {
			return err
		}
		thingsNotToRevisit, err := promptWithDefault(reader, "Things Not To Revisit", existing["Things Not To Revisit"])
		if err != nil {
			return err
		}

		data = SnapshotData{
			Task:               task,
			Objective:          objective,
			CurrentState:       currentState,
			Decisions:          decisions,
			RelevantFiles:      relevantFiles,
			LastStep:           lastStep,
			NextStep:           nextStep,
			ActiveIssues:       activeIssues,
			Constraints:        constraints,
			ThingsNotToRevisit: thingsNotToRevisit,
		}
	}

	outputName := filestore.NewSnapshotName()
	outputPath := filepath.Join(taskDir, outputName)
	content := buildSnapshotMarkdown(data)

	return confirmAndSave(task, buildSnapshotSummary(&data), autoConfirm, func() error {
		if err := filestore.AtomicWrite(outputPath, []byte(content)); err != nil {
			return fmt.Errorf("cannot write snapshot: %w", err)
		}
		if err := commitTaskWrite(project, task, "capture", "snapshot refreshed", []string{taskFile(project, task, outputName)}); err != nil {
			return err
		}
		return nil
	}, fmt.Sprintf("Snapshot updated: %s", outputPath))
}
