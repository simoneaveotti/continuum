
<!-- CONTINUUM:START -->
<!-- do not remove or modify the %[1]s placeholders — they are replaced by ctx agent install -->
<!-- CONTINUUM:BOOTSTRAP_VERSION %[2]s -->
This workspace uses Continuum for agent continuity.

Default Continuum project: %[1]s

Continuum instructions are an execution protocol, not background context.
Follow the flow below, even for simple requests.

Continuum storage lives outside this workspace (`~/.ctx/`, or `CONTINUUM_PATH`).
You must treat `ctx` as the only bridge to that storage.
Do not read from or write to Continuum storage directly.

---

## 0. Session Start (CRITICAL)

Run before any response, inspection, or action:

    ctx context --project=%[1]s

If compact context is insufficient, run full context without asking:

    ctx context --project=%[1]s  # omit --compact for full context

If `ctx` fails:
- continue normally, say context couldn't load
- ask only for minimum missing context
- do not invent state or act as if context was loaded

---

## 1. Tasks

Check available tasks immediately after context:

    ctx list --project=%[1]s

- existing open task matches → use it
- clearly distinct work → start one: `CONTINUUM_AGENT=claude ctx task start <task> --project=%[1]s`
- never start a task without checking first
- never ask the user to restate info already in context

---

## 1.5 Skills (optional)

Check available skills for cross-project reuse:

    ctx skill list
    ctx skill show index    # overview of available skills
    ctx skill show <name>   # full content

Save, create or update:

    ctx skill save <name> [--description=<text>] --yes <<'EOF'
    # <Skill Title>
    ...
    EOF

If the user references a skill by name, try loading it autonomously.

---

## 2. During Work

- continue naturally; don't call Continuum commands repeatedly
- don't persist during exploration or incomplete reasoning
- always self-identify with `CONTINUUM_AGENT=<stable-name>` on write operations
- if commands fail, account for the failure and continue with minimum context
- if instructions conflict, preserve storage safety and lifecycle correctness first

---

## Lifecycle

| Situation | Command |
|---|---|
| Progress made, task open | `CONTINUUM_AGENT=<stable-name> ctx capture <task> --project=%[1]s --type=state --yes` |
| Work paused, continue later | `CONTINUUM_AGENT=<stable-name> ctx handoff <task> --project=%[1]s --yes` |
| Work finished | `CONTINUUM_AGENT=<stable-name> ctx task close <task> --project=%[1]s` |

`ctx handoff` is not `ctx task close`.

### Capture types

- `--type=state` (default): objective, current state, decisions, next step, constraints, active issues
- `--type=proposal`: proposed approach + rationale + review needed
- `--type=request`: review/input request to another agent or user
- `--type=response`: reply to a proposal or request
- `--type=decision`: accepted decision that should be easy to find

Use `ctx artifact list/show` to inspect; `ctx resolve` to clear handled ones.

### Rules

- run `ctx capture` immediately on meaningful progress — do not ask, save, and report "Continuum updated."
- NEVER ask permission before capture/handoff/close/task-switch
- capture on validated progress, not only finished work (code changes, UX changes, decisions, test results, checkpoints)
- don't use `ctx handoff` as substitute for closing a completed task
- if work shifts to a different task, create or switch tasks autonomously
- prefer small, concrete steps over broad redesigns
- do not invent architecture, replace context with generic plans, or force structured input

### State capture template

    CONTINUUM_AGENT=<stable-name> ctx capture <task> --project=%[1]s --type=state --yes <<'EOF'
    ## Objective
    <one line>

    ## Current State
    - <what is done or in progress>

    ## Decisions (Locked)
    - <decision made and why>

    ## Next Step
    - <immediate next action>

    ## Constraints
    - <any hard constraints>

    ## Active Issues
    - <open problems>
    EOF

### Handoff template

    CONTINUUM_AGENT=<stable-name> ctx handoff <task> --project=%[1]s --yes <<'EOF'
    ## Objective
    <one line>

    ## What Was Done
    - <completed work>

    ## Current State
    - <current status>

    ## Next Recommended Step
    - <best next action>

    ## Unresolved Questions
    - <open questions>
    EOF

### Snapshot refresh (manual maintenance, on request only)

    CONTINUUM_AGENT=<stable-name> ctx snapshot refresh <task> --project=%[1]s --yes <<'EOF'
    ## Objective
    <one line>

    ## Current State
    - <current status>

    ## Next Step
    - <immediate next action>

    ## Active Issues
    - <open problems>
    EOF

---

## Project Onboarding (rare, appendix)

If the user asks to onboard the current project, the instruction is already sufficient.
Do not ask them to craft a special prompt.

Analyze the codebase, prepare `project.md` content, and save through:

    CONTINUUM_AGENT=<stable-name> ctx project onboard %[1]s --yes <<'EOF'
    <project markdown content>
    EOF

Use `--yes` (agent-driven); add `--force` only when replacing existing real content.
Do not read or write to storage files directly.
<!-- CONTINUUM:END -->
