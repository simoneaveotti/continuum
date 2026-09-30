package context

import (
	"fmt"
	"sort"
	"strings"

	"continuum/internal/parse"
)

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
	styles := mergeWorkingStyles(ctx.Profile, ctx.Project)
	if len(styles) > 0 {
		lines = append(lines, "WORKING STYLE:")
		for i, s := range styles {
			if len(styles) > 8 && i >= 8 {
				// Keep the context bounded while leaving room for project rules.
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
			if line != "" && line != "..." && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "<!--") && !strings.HasPrefix(line, "-->") {
				items = append(items, line)
			}
		}
	}
	return items
}

// mergeWorkingStyles keeps project-specific rules first, then appends global
// preferences without repeating an identical item.
func mergeWorkingStyles(profile, project string) []string {
	styles := append(extractWorkingStyle(project), extractWorkingStyle(profile)...)
	seen := make(map[string]struct{}, len(styles))
	result := make([]string, 0, len(styles))
	for _, style := range styles {
		key := strings.ToLower(strings.TrimSpace(style))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, style)
	}
	return result
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
