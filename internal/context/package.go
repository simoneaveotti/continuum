package context

import (
	"strings"

	"continuum/internal/parse"
)

func isEmpty(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || s == "..."
}

func cleanValue(value string) string {
	return parse.CleanValue(value)
}

func writeSection(b *strings.Builder, title, value string) {
	value = cleanValue(value)
	if isEmpty(value) {
		return
	}
	b.WriteString(title)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteString("\n\n")
}

func BuildPromptOnlyPackage(ctx *ContextData) string {
	snap := ParseSections(ctx.Snapshot)
	handoff := ParseSections(ctx.Handoff)

	var b strings.Builder

	b.WriteString("=== OBJECTIVE ===\n\n")
	writeSection(&b, "Goal", snap["Objective"])
	writeSection(&b, "Current State", snap["Current State"])
	writeSection(&b, "Next Step", snap["Next Step"])
	writeSection(&b, "Decisions", snap["Decisions (Locked)"])
	writeSection(&b, "Issues", snap["Active Issues"])
	writeSection(&b, "Constraints", snap["Constraints"])
	writeSection(&b, "Files", snap["Relevant Files"])

	b.WriteString("=== LAST SESSION ===\n\n")
	writeSection(&b, "What Was Done", handoff["What Was Done"])
	writeSection(&b, "Risks", handoff["Risks / Caveats"])
	writeSection(&b, "Agent Notes", handoff["Agent Notes"])

	b.WriteString("=== GUIDANCE ===\n\n")
	writeSection(&b, "Next Step", handoff["Next Recommended Step"])
	writeSection(&b, "First Action", handoff["Suggested First Action"])

	b.WriteString("=== UNCERTAINTY ===\n\n")
	writeSection(&b, "Questions", handoff["Unresolved Questions"])
	writeSection(&b, "Assumptions", handoff["Assumptions To Validate"])
	writeSection(&b, "Might Be Wrong", handoff["Things That Might Be Wrong"])
	writeSection(&b, "Missing", handoff["Missing Context"])
	writeSection(&b, "Ask Before Proceeding", handoff["Ask Before Proceeding If"])

	b.WriteString("=== INSTRUCTIONS ===\n\n")
	b.WriteString(`Continue this task.
- Do not repeat known context
- Respect locked decisions
- Validate assumptions before acting
- Start from the next step
- Ask if context is unclear
`)

	return b.String()
}
