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

type HandoffData struct {
	Task                  string
	Objective             string
	WhatWasDone           string
	CurrentState          string
	DecisionsConfirmed    string
	RelevantFiles         string
	RisksCaveats          string
	NextRecommendedStep   string
	AgentNotes            string
	UnresolvedQuestions   string
	AssumptionsToValidate string
	ThingsMightBeWrong    string
	MissingContext        string
	AskBeforeProceedingIf string
	SuggestedFirstAction  string
}

func parseHandoffFromMarkdown(task string, content string) *HandoffData {
	sections := parseSections(content)
	return &HandoffData{
		Task:                  task,
		Objective:             cleanPrefill(sections["Objective"]),
		WhatWasDone:           cleanPrefill(sections["What Was Done"]),
		CurrentState:          cleanPrefill(sections["Current State"]),
		DecisionsConfirmed:    cleanPrefill(sections["Decisions Confirmed"]),
		RelevantFiles:         cleanPrefill(sections["Relevant Files"]),
		RisksCaveats:          cleanPrefill(sections["Risks / Caveats"]),
		NextRecommendedStep:   cleanPrefill(sections["Next Recommended Step"]),
		AgentNotes:            cleanPrefill(sections["Agent Notes"]),
		UnresolvedQuestions:   cleanPrefill(sections["Unresolved Questions"]),
		AssumptionsToValidate: cleanPrefill(sections["Assumptions To Validate"]),
		ThingsMightBeWrong:    cleanPrefill(sections["Things That Might Be Wrong"]),
		MissingContext:        cleanPrefill(sections["Missing Context"]),
		AskBeforeProceedingIf: cleanPrefill(sections["Ask Before Proceeding If"]),
		SuggestedFirstAction:  cleanPrefill(sections["Suggested First Action"]),
	}
}

func buildHandoffSummary(h *HandoffData) string {
	var lines []string
	if h.Objective != "" {
		lines = append(lines, fmt.Sprintf("Objective: %s", h.Objective))
	}
	if h.WhatWasDone != "" {
		lines = append(lines, fmt.Sprintf("What Was Done: %s", h.WhatWasDone))
	}
	if h.CurrentState != "" {
		lines = append(lines, fmt.Sprintf("Current State: %s", h.CurrentState))
	}
	if h.NextRecommendedStep != "" {
		lines = append(lines, fmt.Sprintf("Next Recommended Step: %s", h.NextRecommendedStep))
	}
	if h.UnresolvedQuestions != "" {
		lines = append(lines, fmt.Sprintf("Unresolved Questions: %s", h.UnresolvedQuestions))
	}
	if len(lines) == 0 {
		return "(no content)"
	}
	return strings.Join(lines, "\n")
}

func buildHandoffMarkdown(h HandoffData) string {
	now := time.Now().Format("2006-01-02 15:04:05 MST")

	return fmt.Sprintf(`# TASK HANDOFF

## Task
%s

## Objective
%s

## What Was Done
- %s

## Current State
- %s

## Decisions Confirmed
- %s

## Relevant Files
- %s

## Risks / Caveats
- %s

## Next Recommended Step
- %s

## Agent Notes
- %s

# SURVEY FOR NEXT AGENT

## Unresolved Questions
- %s

## Assumptions To Validate
- %s

## Things That Might Be Wrong
- %s

## Missing Context
- %s

## Ask Before Proceeding If
- %s

## Suggested First Action
- %s

## Last Updated
%s
`,
		h.Task,
		h.Objective,
		h.WhatWasDone,
		h.CurrentState,
		h.DecisionsConfirmed,
		h.RelevantFiles,
		h.RisksCaveats,
		h.NextRecommendedStep,
		h.AgentNotes,
		h.UnresolvedQuestions,
		h.AssumptionsToValidate,
		h.ThingsMightBeWrong,
		h.MissingContext,
		h.AskBeforeProceedingIf,
		h.SuggestedFirstAction,
		now,
	)
}

func loadExistingHandoff(task, project string) map[string]string {
	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	path, _, err := filestore.LatestHandoff(taskDir)
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

func Handoff(task, project string, autoConfirm bool) error {
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

	var data HandoffData
	if piped {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
		data = *parseHandoffFromMarkdown(task, string(raw))
	} else {
		reader := bufio.NewReader(os.Stdin)
		existing := loadExistingHandoff(task, project)

		fmt.Printf("Updating handoff for task '%s'\n", task)

		objective, err := promptWithDefault(reader, "Objective", existing["Objective"])
		if err != nil {
			return err
		}
		whatWasDone, err := promptWithDefault(reader, "What Was Done", existing["What Was Done"])
		if err != nil {
			return err
		}
		currentState, err := promptWithDefault(reader, "Current State", existing["Current State"])
		if err != nil {
			return err
		}
		decisionsConfirmed, err := promptWithDefault(reader, "Decisions Confirmed", existing["Decisions Confirmed"])
		if err != nil {
			return err
		}
		relevantFiles, err := promptWithDefault(reader, "Relevant Files", existing["Relevant Files"])
		if err != nil {
			return err
		}
		risksCaveats, err := promptWithDefault(reader, "Risks / Caveats", existing["Risks / Caveats"])
		if err != nil {
			return err
		}
		nextRecommendedStep, err := promptWithDefault(reader, "Next Recommended Step", existing["Next Recommended Step"])
		if err != nil {
			return err
		}
		agentNotes, err := promptWithDefault(reader, "Agent Notes", existing["Agent Notes"])
		if err != nil {
			return err
		}
		unresolvedQuestions, err := promptWithDefault(reader, "Unresolved Questions", existing["Unresolved Questions"])
		if err != nil {
			return err
		}
		assumptionsToValidate, err := promptWithDefault(reader, "Assumptions To Validate", existing["Assumptions To Validate"])
		if err != nil {
			return err
		}
		thingsMightBeWrong, err := promptWithDefault(reader, "Things That Might Be Wrong", existing["Things That Might Be Wrong"])
		if err != nil {
			return err
		}
		missingContext, err := promptWithDefault(reader, "Missing Context", existing["Missing Context"])
		if err != nil {
			return err
		}
		askBeforeProceedingIf, err := promptWithDefault(reader, "Ask Before Proceeding If", existing["Ask Before Proceeding If"])
		if err != nil {
			return err
		}
		suggestedFirstAction, err := promptWithDefault(reader, "Suggested First Action", existing["Suggested First Action"])
		if err != nil {
			return err
		}

		data = HandoffData{
			Task:                  task,
			Objective:             objective,
			WhatWasDone:           whatWasDone,
			CurrentState:          currentState,
			DecisionsConfirmed:    decisionsConfirmed,
			RelevantFiles:         relevantFiles,
			RisksCaveats:          risksCaveats,
			NextRecommendedStep:   nextRecommendedStep,
			AgentNotes:            agentNotes,
			UnresolvedQuestions:   unresolvedQuestions,
			AssumptionsToValidate: assumptionsToValidate,
			ThingsMightBeWrong:    thingsMightBeWrong,
			MissingContext:        missingContext,
			AskBeforeProceedingIf: askBeforeProceedingIf,
			SuggestedFirstAction:  suggestedFirstAction,
		}
	}

	outputName := filestore.NewHandoffName()
	outputPath := filepath.Join(taskDir, outputName)
	content := buildHandoffMarkdown(data)

	return confirmAndSave(task, buildHandoffSummary(&data), autoConfirm, func() error {
		if err := filestore.AtomicWrite(outputPath, []byte(content)); err != nil {
			return fmt.Errorf("cannot write handoff: %w", err)
		}
		if err := commitTaskWrite(project, task, "handoff", "handoff created", []string{taskFile(project, task, outputName)}); err != nil {
			return err
		}
		return nil
	}, fmt.Sprintf("Handoff updated: %s", outputPath))
}
