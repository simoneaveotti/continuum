package context

import (
	"strings"
)

func BuildPromptOnlyPackage(ctx *ContextData) string {
	snap := ParseSections(ctx.Snapshot)
	handoff := ParseSections(ctx.Handoff)

	var b strings.Builder

	b.WriteString("=== CONTEXT ===\n\n")

	b.WriteString("OBJECTIVE:\n")
	b.WriteString(snap["Objective"])
	b.WriteString("\n\n")

	b.WriteString("CURRENT STATE:\n")
	b.WriteString(snap["Current State"])
	b.WriteString("\n\n")

	b.WriteString("NEXT STEP:\n")
	b.WriteString(snap["Next Step"])
	b.WriteString("\n\n")

	b.WriteString("ISSUES:\n")
	b.WriteString(snap["Active Issues"])
	b.WriteString("\n\n")

	b.WriteString("DECISIONS:\n")
	b.WriteString(snap["Decisions (Locked)"])
	b.WriteString("\n\n")

	b.WriteString("=== HANDOFF ===\n\n")

	b.WriteString("WHAT WAS DONE:\n")
	b.WriteString(handoff["What Was Done"])
	b.WriteString("\n\n")

	b.WriteString("NEXT:\n")
	b.WriteString(handoff["Next Recommended Step"])
	b.WriteString("\n\n")

	b.WriteString("=== SURVEY ===\n\n")

	b.WriteString("QUESTIONS:\n")
	b.WriteString(handoff["Unresolved Questions"])
	b.WriteString("\n\n")

	b.WriteString("ASSUMPTIONS:\n")
	b.WriteString(handoff["Assumptions To Validate"])
	b.WriteString("\n\n")

	b.WriteString("MISSING:\n")
	b.WriteString(handoff["Missing Context"])
	b.WriteString("\n\n")

	b.WriteString("=== PROMPT ===\n")
	b.WriteString(`Continue the task using the context above.

Rules:
- do not repeat known context
- respect locked decisions
- validate assumptions before acting
- proceed from the next step

If something is unclear:
- use questions and missing context sections
`)

	return b.String()
}
