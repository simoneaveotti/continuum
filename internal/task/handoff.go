package task

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func promptField(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
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

func loadExistingHandoff(task string) map[string]string {
	path := filepath.Join(".continuum", "tasks", task, "handoff.md")

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

func Handoff(task string) error {
	if task == "" {
		return fmt.Errorf("task name is required")
	}

	taskDir := filepath.Join(".continuum", "tasks", task)
	if _, err := os.Stat(taskDir); err != nil {
		return fmt.Errorf("task directory not found: %s", taskDir)
	}

	reader := bufio.NewReader(os.Stdin)

	existing := loadExistingHandoff(task)

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

	data := HandoffData{
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

	outputPath := filepath.Join(taskDir, "handoff.md")
	content := buildHandoffMarkdown(data)

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("cannot write handoff.md: %w", err)
	}

	fmt.Printf("Handoff updated: %s\n", outputPath)
	return nil
}
