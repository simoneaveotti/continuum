# ADR 006: In-Scope Agent Bootstrap

## Status

Accepted

## Context

Continuum must work with any coding agent, but agents:

- Cannot reliably access files outside their current workspace
- Cannot discover hidden directories like `.ctx/`
- May not support external tools (CLI, MCP, etc.)
- May be local models (Ollama, etc.)

Continuum must:
- Be portable across machines
- Work without global configuration
- Work in minimal setups
- Not modify project repositories

## Decision

**In-scope bootstrap model**: explicitly inject context into the agent's instruction scope.

### 1. Bootstrap Command

```bash
ctx agent install
```

Injects Continuum bootstrap block into common agent instruction files:
- `AGENTS.md`
- `CLAUDE.md`
- `agent.md`

Uses markers:
```markdown
<!-- CONTINUUM:START -->
[bootstrap content]
<!-- CONTINUUM:END -->
```

Idempotent: safe to run multiple times, no duplication.

When bootstrap templates change and existing agent files must be updated in place, run:

```bash
ctx agent install --project=<project> --force
```

This re-injects the current bootstrap block instead of leaving an older installed block untouched.

### 2. Bootstrap Block Content

The injected block instructs the agent to:

- Run `ctx context --project=<project>` as the first action of every session
- Not ask user to restate known information already present in Continuum context
- Work through natural conversation
- Use `ctx list` immediately after loading project context
- Persist routine progress on the active task autonomously with `ctx capture`
- Use piped markdown plus `--yes` in non-interactive environments when needed
- Preserve conversation flow by handling Continuum actions autonomously, including `handoff`, `snapshot refresh`, and task creation or switching when clearly needed
- NOT invent new architecture

Example:
```markdown
<!-- CONTINUUM:START -->
## Session Start (MANDATORY)

Run this first, before writing a single word of response and before any other action:

    ctx context --project=my-project

## Task Check

Immediately after loading context, check available tasks:

    ctx list --project=my-project

## After Meaningful Progress

Run `ctx capture` immediately when progress is worth recording. Do not ask the user first.
After saving, report in one line: "Continuum updated."

Rules:
- run `ctx capture` immediately on meaningful progress
- do not ask the user for permission before `ctx capture`, `ctx handoff`, `ctx snapshot refresh`, or task creation/switching; preserve conversation flow and act autonomously
<!-- CONTINUUM:END -->
```

### 3. No Global Dependencies

The system does NOT rely on:
- Global configuration paths (`~/.ctx`)
- Automatic file discovery
- Implicit behavior

### 4. Context Injection

When starting work, the agent instruction file must direct the agent to retrieve context itself via:
- `ctx context --project=<project>`

The bootstrap must assume the agent can run shell commands in the workspace, but cannot directly read or write `~/.ctx/`.

### 5. Persistence Model

Routine persistence is autonomous. Agents should persist or transition task state without interrupting the user:

- Normal active-task progress:
  - use `ctx capture`
  - should happen autonomously
  - should not require repetitive user approval prompts
- Continuity and task-management actions:
  - `ctx handoff`
  - `ctx snapshot refresh`
  - task creation / switching
  - should also be handled autonomously when the correct action is clear from context
- Ambiguous or unsafe state updates:
  - agents should resolve the ambiguity conservatively, but should not default to asking the user for permission to use the tool itself

### 6. MCP (Future)

Optional MCP integration may be added later, but is NOT required for baseline.

## Consequences

### Positive
- Fully portable across environments
- Works with any agent that reads instruction files
- No global configuration needed
- Predictable, explicit onboarding
- Works with local models
- Stronger bootstrap compliance across agents because the first command is mandatory and operationally explicit
- Lower friction for routine continuity updates on the active task

### Negative
- Requires explicit bootstrap step
- Consumes tokens for context injection
- Must maintain bootstrap block format
- Depends on agent reading the file
- Requires careful wording to avoid contradictory rules around autonomous vs explicit persistence

## Notes

**Continuum does not rely on agent-side discovery.**

Instead:
- Bootstrap is explicitly injected
- Context is explicitly provided
- Onboarding is explicit and repeatable

This ensures reliability regardless of agent capabilities.
