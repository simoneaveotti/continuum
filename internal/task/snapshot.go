package task

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"
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

func loadExistingSnapshot(task string) map[string]string {
	path := filepath.Join(".continuum", "tasks", task, "snapshot.md")

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

func SnapshotRefresh(task string) error {
	if task == "" {
		return fmt.Errorf("task name is required")
	}

	taskDir := filepath.Join(".continuum", "tasks", task)
	if _, err := os.Stat(taskDir); err != nil {
		return fmt.Errorf("task directory not found: %s", taskDir)
	}

	reader := bufio.NewReader(os.Stdin)

	existing := loadExistingSnapshot(task)

	fmt.Printf("Updating snapshot for task '%s'\n", task)

	objective, _ := promptWithDefault(reader, "Objective", existing["Objective"])
	currentState, _ := promptWithDefault(reader, "Current State", existing["Current State"])
	decisions, _ := promptWithDefault(reader, "Decisions (Locked)", existing["Decisions (Locked)"])
	relevantFiles, _ := promptWithDefault(reader, "Relevant Files", existing["Relevant Files"])
	lastStep, _ := promptWithDefault(reader, "Last Step Completed", existing["Last Step Completed"])
	nextStep, _ := promptWithDefault(reader, "Next Step", existing["Next Step"])
	activeIssues, _ := promptWithDefault(reader, "Active Issues", existing["Active Issues"])
	constraints, _ := promptWithDefault(reader, "Constraints", existing["Constraints"])
	thingsNotToRevisit, _ := promptWithDefault(reader, "Things Not To Revisit", existing["Things Not To Revisit"])

	data := SnapshotData{
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

	outputPath := filepath.Join(taskDir, "snapshot.md")
	content := buildSnapshotMarkdown(data)

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("cannot write snapshot.md: %w", err)
	}

	fmt.Printf("Snapshot updated: %s\n", outputPath)
	return nil
}
