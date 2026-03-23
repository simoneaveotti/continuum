package render

import (
	"fmt"

	"continuum/internal/context"
)

func PrintFull(ctx *context.ContextData) {
	fmt.Println("=== PROFILE CONTEXT ===")
	fmt.Println(ctx.Profile)

	fmt.Println()
	fmt.Println("=== PROJECT CONTEXT ===")
	fmt.Println(ctx.Project)

	fmt.Println()
	fmt.Println("=== TASK SNAPSHOT ===")
	fmt.Println(ctx.Snapshot)

	fmt.Println()
	fmt.Println("=== TASK HANDOFF ===")
	fmt.Println(ctx.Handoff)

	fmt.Println()
	fmt.Println("=== OPERATIONAL PROMPT ===")
	fmt.Print(`Continue this task using the context above.

Rules:
- Respect decisions already taken
- Do not re-open settled choices without reason
- Validate assumptions before proceeding
- Start from the next step or suggested action
`)
}

func PrintPromptOnly(ctx *context.ContextData) {
	fmt.Print(context.BuildPromptOnlyPackage(ctx))
}
