package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"continuum/internal/filestore"
	"continuum/internal/parse"
	"continuum/internal/setup"
	"continuum/internal/task"
)

type CollaborationArtifacts struct {
	ProposalCount  int
	RequestCount   int
	LatestProposal string
	LatestRequest  string
	LatestResponse string
	LatestDecision string
}

func LoadFullContext(project string) (*ContextData, error) {
	if err := setup.ValidateProjectName(project); err != nil {
		return nil, err
	}

	base := setup.ContinuumPath()

	if err := setup.PullLatest(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	profile, err := os.ReadFile(filepath.Join(base, "profile.md"))
	if err != nil {
		if os.IsNotExist(err) {
			profile = []byte("# Profile\n\n(No profile set)")
		} else {
			return nil, fmt.Errorf("cannot read profile: %w", err)
		}
	}

	projectData, err := os.ReadFile(filepath.Join(base, "projects", project, "project.md"))
	if err != nil {
		if os.IsNotExist(err) {
			projectData = []byte("# Project\n\n(No project context)")
		} else {
			return nil, fmt.Errorf("cannot read project: %w", err)
		}
	}

	tasksDir := filepath.Join(base, "projects", project, "tasks")
	tasks := []string{}
	if _, err := os.Stat(tasksDir); err == nil {
		taskInfos, err := task.ListWithStatus(project, string(task.StatusActive))
		if err != nil {
			return nil, err
		}
		for _, info := range taskInfos {
			tasks = append(tasks, info.Name)
		}
	}

	taskContexts := make(map[string]*ContextData)
	for _, t := range tasks {
		if tc, err := load(t, project, false); err == nil {
			taskContexts[t] = tc
		}
	}

	return &ContextData{
		Profile:      string(profile),
		Project:      string(projectData),
		Snapshot:     strings.Join(tasks, "\n"),
		Handoff:      "",
		Unsynced:     setup.UnsyncedCommits(),
		TaskContexts: taskContexts,
	}, nil
}

func BuildContextPackage(ctx *ContextData, task, project string) string {
	var lines []string

	// PROJECT
	lines = append(lines, fmt.Sprintf("PROJECT: %s", project))
	summary := parse.ExtractField(ctx.Project, "summary")
	if summary == "" || summary == "..." {
		summary = project
	}
	// Take only the first line (in case there are newlines)
	if idx := strings.Index(summary, "\n"); idx >= 0 {
		summary = summary[:idx]
	}
	// Trim spaces but do not truncate
	summary = strings.TrimSpace(summary)
	if summary != "" {
		lines = append(lines, "")
		lines = append(lines, summary)
	}

	// STACK - extract core tech names
	stack := extractStackCore(ctx.Project)
	if stack != "" {
		lines = append(lines, fmt.Sprintf("STACK: %s", stack))
	}

	if len(ctx.Unsynced) > 0 {
		lines = append(lines, fmt.Sprintf("UNSYNCED: %d commit(s) pending upload", len(ctx.Unsynced)))
	}

	// CONSTRAINTS - extract as-is, cap at 6
	constraints := extractConstraints(ctx.Project)
	if len(constraints) > 0 {
		limit := len(constraints)
		if limit > 6 {
			limit = 6
		}
		lines = append(lines, "CONSTRAINTS:")
		for i := 0; i < limit; i++ {
			lines = append(lines, fmt.Sprintf("- %s", constraints[i]))
		}
	}

	// WORKING STYLE - extract from profile
	styles := extractWorkingStyle(ctx.Profile)
	if len(styles) > 0 {
		lines = append(lines, "WORKING STYLE:")
		for i, s := range styles {
			if len(styles) > 4 && i >= 4 {
				// Only show first 4
				break
			}
			lines = append(lines, fmt.Sprintf("- %s", s))
		}
	}

	// TASK SPECIFIC SECTIONS
	if task != "" {
		lines = append(lines, fmt.Sprintf("CURRENT FOCUS: %s", task))
		snapshot := ctx.Snapshot
		handoff := ctx.Handoff
		snapshotName := ctx.SnapshotName

		if ctx.TaskContexts != nil {
			if taskCtx, ok := ctx.TaskContexts[task]; ok {
				snapshot = taskCtx.Snapshot
				handoff = taskCtx.Handoff
				snapshotName = taskCtx.SnapshotName
			}
		}

		if snapshot != "" || handoff != "" {
			// OBJECTIVE
			objective := parse.ExtractField(snapshot, "objective")
			if objective == "" || objective == "..." {
				objective = "not yet defined"
			}
			lines = append(lines, fmt.Sprintf("OBJECTIVE: %s", objective))

			// CURRENT STATE
			state := extractStateSimple(snapshot)
			if state != "" {
				lines = append(lines, fmt.Sprintf("CURRENT STATE: %s", state))
			} else {
				lines = append(lines, "CURRENT STATE: not yet defined")
			}

			// NEXT STEP
			nextStep := parse.ExtractField(snapshot, "next step")
			if nextStep == "" || nextStep == "..." {
				nextStep = "not yet defined"
			}
			lines = append(lines, fmt.Sprintf("NEXT STEP: %s", nextStep))
		} else {
			lines = append(lines, "OBJECTIVE: not yet defined")
			lines = append(lines, "CURRENT STATE: not yet defined")
			lines = append(lines, "NEXT STEP: not yet defined")
		}
		lines = appendCollaboration(lines, task, project)
		if snapshotName != "" {
			lines = append(lines, fmt.Sprintf("source snapshot: %s", snapshotName))
		}
	} else if ctx.TaskContexts != nil && len(ctx.TaskContexts) > 0 {
		// If there is exactly one task, treat it as the implicit current focus.
		if len(ctx.TaskContexts) == 1 {
			for onlyTask, taskCtx := range ctx.TaskContexts {
				lines = append(lines, fmt.Sprintf("CURRENT FOCUS: %s", onlyTask))

				objective := parse.ExtractField(taskCtx.Snapshot, "objective")
				if objective == "" || objective == "..." {
					objective = "not yet defined"
				}
				lines = append(lines, fmt.Sprintf("OBJECTIVE: %s", objective))

				state := extractStateSimple(taskCtx.Snapshot)
				if state != "" {
					lines = append(lines, fmt.Sprintf("CURRENT STATE: %s", state))
				} else {
					lines = append(lines, "CURRENT STATE: not yet defined")
				}

				nextStep := parse.ExtractField(taskCtx.Snapshot, "next step")
				if nextStep == "" || nextStep == "..." {
					nextStep = "not yet defined"
				}
				lines = append(lines, fmt.Sprintf("NEXT STEP: %s", nextStep))
				lines = appendCollaboration(lines, onlyTask, project)
				if taskCtx.SnapshotName != "" {
					lines = append(lines, fmt.Sprintf("source snapshot: %s", taskCtx.SnapshotName))
				}
				return strings.Join(lines, "\n")
			}
		}

		// No specific task, show available tasks
		taskNames := []string{}
		for t := range ctx.TaskContexts {
			taskNames = append(taskNames, t)
		}
		if len(taskNames) > 0 {
			if len(taskNames) > 3 {
				lines = append(lines, fmt.Sprintf("CURRENT FOCUS: not yet defined (available: %s)", strings.Join(taskNames[:3], ", ")))
			} else {
				lines = append(lines, fmt.Sprintf("CURRENT FOCUS: not yet defined (available: %s)", strings.Join(taskNames, ", ")))
			}
		} else {
			lines = append(lines, "CURRENT FOCUS: not yet defined")
		}
		lines = append(lines, "OBJECTIVE: not yet defined")
		lines = append(lines, "CURRENT STATE: not yet defined")
		lines = append(lines, "NEXT STEP: not yet defined")
	} else {
		// No tasks at all
		lines = append(lines, "CURRENT FOCUS: not yet defined")
		lines = append(lines, "OBJECTIVE: not yet defined")
		lines = append(lines, "CURRENT STATE: not yet defined")
		lines = append(lines, "NEXT STEP: not yet defined")
	}

	return strings.Join(lines, "\n")
}

func BuildCompactContextPackage(ctx *ContextData, taskName, project string) string {
	snapshot, handoff, snapshotName, focus := resolveFocus(ctx, taskName)
	var lines []string

	header := "PRJ:" + project
	if focus != "" {
		header += " FOCUS:" + focus
	}
	lines = append(lines, header)
	if focus == "" && ctx.TaskContexts != nil && len(ctx.TaskContexts) > 0 {
		lines = append(lines, "TASKS:"+compactTaskList(ctx.TaskContexts, 5))
	}

	appendCompactField := func(key, value string) {
		value = cleanCompactValue(value)
		if value != "" {
			lines = append(lines, key+":"+value)
		}
	}

	if len(ctx.Unsynced) > 0 {
		appendCompactField("UNSYNCED", fmt.Sprintf("%d commit(s) pending upload", len(ctx.Unsynced)))
	}
	appendCompactField("OBJ", parse.ExtractField(snapshot, "objective"))
	appendCompactField("STATE", extractStateSimple(snapshot))
	appendCompactField("NEXT", parse.ExtractField(snapshot, "next step"))
	appendCompactField("ISSUES", parse.ExtractField(snapshot, "active issues"))
	appendCompactField("DECIDED", compactDecisions(snapshot, ctx.Project))
	appendCompactField("LAST", parse.ExtractField(handoff, "what was done"))
	if focus != "" {
		appendCompactCollaboration(&lines, focus, project)
	}
	appendCompactField("SRC", snapshotName)

	if len(lines) == 1 && focus != "" {
		lines = append(lines, "STATE:no snapshot yet")
		lines = append(lines, "NEXT:capture initial state")
	}

	return strings.Join(lines, "\n")
}

func compactTaskList(tasks map[string]*ContextData, limit int) string {
	names := make([]string, 0, len(tasks))
	for name := range tasks {
		names = append(names, name)
	}
	sort.Strings(names)
	if limit > 0 && len(names) > limit {
		return strings.Join(names[:limit], " | ") + fmt.Sprintf(" | +%d more", len(names)-limit)
	}
	return strings.Join(names, " | ")
}

func resolveFocus(ctx *ContextData, taskName string) (snapshot, handoff, snapshotName, focus string) {
	if taskName != "" {
		focus = taskName
		snapshot = ctx.Snapshot
		handoff = ctx.Handoff
		snapshotName = ctx.SnapshotName
		if ctx.TaskContexts != nil {
			if taskCtx, ok := ctx.TaskContexts[taskName]; ok {
				snapshot = taskCtx.Snapshot
				handoff = taskCtx.Handoff
				snapshotName = taskCtx.SnapshotName
			}
		}
		return snapshot, handoff, snapshotName, focus
	}
	if ctx.TaskContexts != nil && len(ctx.TaskContexts) == 1 {
		for onlyTask, taskCtx := range ctx.TaskContexts {
			return taskCtx.Snapshot, taskCtx.Handoff, taskCtx.SnapshotName, onlyTask
		}
	}
	return "", "", "", ""
}

func appendCompactCollaboration(lines *[]string, taskName, project string) {
	artifacts, err := LoadCollaborationArtifacts(taskName, project)
	if err != nil {
		return
	}
	var parts []string
	if artifacts.ProposalCount > 0 {
		parts = append(parts, fmt.Sprintf("proposals=%d", artifacts.ProposalCount))
	}
	if artifacts.RequestCount > 0 {
		parts = append(parts, fmt.Sprintf("requests=%d", artifacts.RequestCount))
	}
	if len(parts) > 0 {
		*lines = append(*lines, "OPEN:"+strings.Join(parts, " | "))
	}
	if value := cleanCompactValue(artifacts.LatestResponse); value != "" {
		*lines = append(*lines, "RESP:"+value)
	}
	if value := cleanCompactValue(artifacts.LatestDecision); value != "" {
		*lines = append(*lines, "DECISION:"+value)
	}
}

func compactDecisions(snapshot, projectData string) string {
	var decisions []string
	if value := cleanCompactValue(parse.ExtractField(snapshot, "locked decisions")); value != "" {
		decisions = append(decisions, value)
	}
	for _, constraint := range extractConstraints(projectData) {
		if value := cleanCompactValue(constraint); value != "" {
			decisions = append(decisions, value)
		}
		if len(decisions) >= 3 {
			break
		}
	}
	return strings.Join(decisions, " | ")
}

func cleanCompactValue(value string) string {
	value = compactSummary(value)
	lower := strings.ToLower(strings.TrimSpace(value))
	switch lower {
	case "", "...", "none", "not yet defined", "content provided":
		return ""
	default:
		return value
	}
}

func LoadCollaborationArtifacts(taskName, project string) (*CollaborationArtifacts, error) {
	if err := setup.ValidateTaskName(taskName); err != nil {
		return nil, err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return nil, err
	}
	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", taskName)
	artifacts := &CollaborationArtifacts{}

	proposals, err := filestore.AllCapturesOfType(taskDir, filestore.ProposalCapture)
	if err != nil {
		return nil, err
	}
	requests, err := filestore.AllCapturesOfType(taskDir, filestore.RequestCapture)
	if err != nil {
		return nil, err
	}
	artifacts.ProposalCount = len(proposals)
	artifacts.RequestCount = len(requests)
	if len(proposals) > 0 {
		artifacts.LatestProposal = latestArtifactSummary(proposals[len(proposals)-1])
	}
	if len(requests) > 0 {
		artifacts.LatestRequest = latestArtifactSummary(requests[len(requests)-1])
	}
	if path, _, err := filestore.LatestCaptureOfType(taskDir, filestore.ResponseCapture); err == nil && path != "" {
		artifacts.LatestResponse = latestArtifactSummary(path)
	} else if err != nil {
		return nil, err
	}
	if path, _, err := filestore.LatestCaptureOfType(taskDir, filestore.DecisionCapture); err == nil && path != "" {
		artifacts.LatestDecision = latestArtifactSummary(path)
	} else if err != nil {
		return nil, err
	}
	return artifacts, nil
}

func appendCollaboration(lines []string, taskName, project string) []string {
	artifacts, err := LoadCollaborationArtifacts(taskName, project)
	if err != nil {
		return lines
	}
	if artifacts.ProposalCount > 0 {
		lines = append(lines, fmt.Sprintf("OPEN PROPOSALS: %d (latest: %s)", artifacts.ProposalCount, artifacts.LatestProposal))
	}
	if artifacts.RequestCount > 0 {
		lines = append(lines, fmt.Sprintf("OPEN REQUESTS: %d (latest: %s)", artifacts.RequestCount, artifacts.LatestRequest))
	}
	if artifacts.LatestResponse != "" {
		lines = append(lines, fmt.Sprintf("LATEST RESPONSE: %s", artifacts.LatestResponse))
	}
	if artifacts.LatestDecision != "" {
		lines = append(lines, fmt.Sprintf("LATEST DECISION: %s", artifacts.LatestDecision))
	}
	return lines
}

func latestArtifactSummary(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "unreadable artifact"
	}
	content := string(data)
	body := artifactBody(content)
	for _, section := range []string{"decision", "recommendation", "response", "request", "proposal"} {
		if value := parse.ExtractField(body, section); value != "" && value != "..." {
			return compactSummary(value)
		}
	}
	return compactSummary(firstUserArtifactLine(body))
}

func firstUserArtifactLine(content string) string {
	for _, line := range splitLines(content) {
		line = trimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.EqualFold(line, "## Last Updated") {
			break
		}
		return line
	}
	return "content provided"
}

func artifactBody(content string) string {
	lines := splitLines(content)
	for i, line := range lines {
		if strings.EqualFold(trimSpace(line), "## Capture Type") {
			start := i + 1
			for start < len(lines) && trimSpace(lines[start]) != "" {
				start++
			}
			for start < len(lines) && trimSpace(lines[start]) == "" {
				start++
			}
			return strings.Join(lines[start:], "\n")
		}
	}
	return content
}

func compactSummary(value string) string {
	value = trimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")
	for strings.Contains(value, "  ") {
		value = strings.ReplaceAll(value, "  ", " ")
	}
	if value == "" {
		return "content provided"
	}
	if len(value) > 140 {
		return value[:137] + "..."
	}
	return value
}

// Helper functions

func extractStackCore(content string) string {
	items := parse.ExtractBulletList(content, "stack")
	if len(items) == 0 {
		return ""
	}
	var result []string
	seen := make(map[string]bool)
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || item == "..." {
			continue
		}
		// Remove content in parentheses (version notes, descriptions)
		if idx := strings.Index(item, "("); idx >= 0 {
			item = strings.TrimSpace(item[:idx])
		}
		// For "X / Y" format take only the first part as the primary name
		if idx := strings.Index(item, " / "); idx >= 0 {
			item = strings.TrimSpace(item[:idx])
		}
		item = strings.TrimSpace(item)
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	if len(result) > 6 {
		result = result[:6]
	}
	return strings.Join(result, ", ")
}

func extractConstraints(content string) []string {
	items := parse.ExtractBulletList(content, "constraint")
	if len(items) == 0 {
		return []string{}
	}
	var result []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || item == "..." {
			continue
		}
		// Remove leading dash/asterisk
		item = strings.TrimPrefix(item, "-")
		item = strings.TrimPrefix(item, "*")
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func extractWorkingStyle(content string) []string {
	var items []string
	inSection := false
	for _, line := range splitLines(content) {
		line = trimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "## ") || strings.HasPrefix(strings.ToLower(line), "# ") {
			if inSection {
				break
			}
			lower := strings.ToLower(line)
			if strings.Contains(lower, "working") || strings.Contains(lower, "style") ||
				strings.Contains(lower, "preference") || strings.Contains(lower, "rule") {
				inSection = true
				continue
			}
			continue
		}
		if inSection {
			line = stripPrefix(line, "-")
			line = stripPrefix(line, "*")
			line = trimSpace(line)
			if line != "" && line != "..." && !strings.HasPrefix(line, "#") {
				items = append(items, line)
			}
		}
	}
	return items
}

func extractStateSimple(content string) string {
	items := extractBulletPoints(content)
	if len(items) == 0 {
		return ""
	}
	// Take first few items and join them
	var result []string
	for i, item := range items {
		if i >= 2 { // Only take first 2 state items
			break
		}
		item = strings.TrimSpace(item)
		if item == "" || item == "..." {
			continue
		}
		// Remove leading dash/asterisk
		item = strings.TrimPrefix(item, "-")
		item = strings.TrimPrefix(item, "*")
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	if len(result) > 0 {
		return strings.Join(result, " | ")
	}
	return ""
}

func extractBulletPoints(content string) []string {
	var lines []string
	inList := false

	for _, line := range splitLines(content) {
		line = trimSpace(line)

		if strings.HasPrefix(strings.ToLower(line), "## ") || strings.HasPrefix(strings.ToLower(line), "# ") {
			inList = false
			lower := strings.ToLower(line)
			if strings.Contains(lower, "state") || strings.Contains(lower, "current") {
				inList = true
				continue
			}
			continue
		}

		if inList {
			line = stripPrefix(line, "-")
			line = stripPrefix(line, "*")
			line = trimSpace(line)
			if line != "" {
				lines = append(lines, line)
			}
		}
	}

	return lines
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

func stripPrefix(s, prefix string) string {
	return strings.TrimPrefix(s, prefix)
}

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}
