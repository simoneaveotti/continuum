# Continuum Agent Skill

You are working inside Continuum.

You must:
- work through natural conversation
- periodically summarize the current state
- persist meaningful progress with `ctx capture`
- start a new task with `CONTINUUM_AGENT=<stable-name> ctx task start <task> --project=<name>` before recording progress for that line of work
- treat a user request to onboard the project as sufficient instruction to analyze the codebase and save `project.md` through `ctx project onboard <name>`
- use `ctx project onboard <name> --yes` for agent-driven onboarding
- add `--force` only when replacing an existing project context that already contains real content
- treat any implemented code change with user-visible behavior, UX, workflow, data flow, schema, task direction, or verified outcome as meaningful progress
- capture immediately after validated progress is in place; do not wait for full task completion
- use `ctx handoff` only for continuity on tasks that remain open
- use `ctx task close` when a task is actually complete
- identify yourself on Continuum write commands with a stable `CONTINUUM_AGENT` value for the current runtime
- use `ctx` as the only bridge to Continuum storage

You must NOT:
- force structured input
- invent new architecture
- re-ask known information already present in context
- read or write `~/.continuum/` directly
- ask the user to craft a special prompt for project onboarding

When summarizing, use natural language first.

Then internally map to:
- OBJECTIVE
- STATE
- NEXT
- CONSTRAINTS
- ISSUES

Quick lifecycle:

- new line of work: `CONTINUUM_AGENT=<stable-name> ctx task start <task> --project=<name>`, then `ctx capture <task> --project=<name>`
- progress made: `CONTINUUM_AGENT=<stable-name> ctx capture <task> --project=<name>`
- paused but not finished: `CONTINUUM_AGENT=<stable-name> ctx handoff <task> --project=<name>`
- finished: `CONTINUUM_AGENT=<stable-name> ctx task close <task> --project=<name>`

Do not treat `ctx handoff` as equivalent to closing a task.
Do not capture progress for a new task before that task has been started.
Do not wait for task completion before capturing.
A partial but real implementation, a validated UX change, a passed test, or a committed checkpoint is already meaningful progress.
