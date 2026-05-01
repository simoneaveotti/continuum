<!-- CONTINUUM:START -->
<!-- do not remove or modify the %[1]s placeholders — they are replaced by ctx agent install -->
This workspace uses Continuum for agent continuity.

Default Continuum project: %[1]s

Continuum instructions are an execution protocol, not background context.
Follow the states below in order, even for simple requests.

Continuum storage lives outside this workspace.
The default location is `~/.ctx/`, but it may be overridden with `CONTINUUM_PATH`.
You must treat `ctx` as the only bridge to that storage.
Do not read from or write to Continuum storage directly.

---

## State 0: Session Start (CRITICAL)

You MUST run this as a shell/terminal command before writing a response,
inspecting files, or taking any other action. Do not translate `ctx ...` into a
runtime tool call such as `ctx:context`.

    ctx context --project=%[1]s --compact

Do not answer, inspect files, or take action before running this command.

If compact context is insufficient for the requested work, run the full context
without asking the user first:

    ctx context --project=%[1]s

If `ctx` is unavailable or this command fails:
- continue normally
- do not block the session
- say that Continuum context could not be loaded
- ask only for the minimum missing context needed to proceed
- do not invent Continuum state or act as if context was loaded

Use the output as the authoritative source of:
- current objective
- current state
- decisions already made
- active issues
- next step

## State 1: Task Check (CRITICAL)

Immediately after loading context, you MUST check available tasks:

    ctx list --project=%[1]s

Review the list carefully before choosing task state.

- if an existing open task matches the current work, use it for `ctx capture`, `ctx handoff`, or `ctx task close`
- if no suitable task exists and the user begins clearly distinct work, start one:

    CONTINUUM_AGENT=<stable-name> ctx task start <task> --project=%[1]s

Do not start a new task without checking existing tasks first.
Do not capture a new line of work until the correct task exists.
If work clearly belongs to a different task, create or switch tasks autonomously.

Do not ask the user to restate information already present in this context.

---

## State 2: During Work

- continue through natural conversation and normal coding activity
- do not call Continuum commands repeatedly
- do not persist state during exploration or incomplete reasoning
- before any Continuum write operation, identify yourself autonomously by invoking the command with a stable `CONTINUUM_AGENT` value for the current runtime
- do not ask the user to set agent identity manually; the agent should handle this itself when issuing the command
- if a required Continuum command fails, account for that failure in the conversation and continue with the minimum context needed
- if instructions conflict, preserve Continuum storage safety and task lifecycle correctness first

Example pattern:

    CONTINUUM_AGENT=claude ctx capture <task> --project=%[1]s --yes <<'EOF'
    ...
    EOF

---

## Quick Lifecycle

Use the task lifecycle commands explicitly:

- progress made, task remains open: `CONTINUUM_AGENT=<stable-name> ctx capture <task> --project=%[1]s --type=state --yes`
- paused and future continuity needed: `CONTINUUM_AGENT=<stable-name> ctx handoff <task> --project=%[1]s --yes`
- actually finished: `CONTINUUM_AGENT=<stable-name> ctx task close <task> --project=%[1]s`

`ctx handoff` is not `ctx task close`.

---

## Project Onboarding

If the user asks to onboard the current project, that instruction is already sufficient.
Do not ask the user to craft a special onboarding prompt.

When onboarding a project:

- analyze the current codebase and prepare the new `project.md` content
- save it through `ctx project onboard %[1]s`
- do not read from or write to the Continuum storage files directly

Use these flags consistently:

- use `--yes` for agent-driven onboarding so the command does not stop for confirmation
- use `--force` only when replacing an existing `project.md` that already contains real project content
- do not add `--force` for first-time onboarding or when the file is still only the template/placeholder content

Canonical agent flow:

    CONTINUUM_AGENT=<stable-name> ctx project onboard %[1]s --yes <<'EOF'
    <project markdown content>
    EOF

If the command reports that project context already exists and must be replaced, rerun it with `--force --yes`.

---

## After Meaningful Progress

When real progress has been made, compose the current state as markdown and pipe it to `ctx capture`.

`ctx capture` defaults to `--type=state`. Use state captures only for the actual
task state that should drive future `ctx context` output: objective, current state,
next step, constraints, and active issues.

Use typed captures for collaboration artifacts that should not replace task state:

- `--type=proposal` for proposed plans, approaches, or changes that need review
- `--type=request` for explicit review/input requests to another agent or the user
- `--type=response` for replies to a proposal or request
- `--type=decision` for decisions that were accepted and should be easy to find

Inspect collaboration artifacts with `ctx artifact list/show`; clear handled ones from open counts with `ctx resolve`.
Do not put proposals, review requests, responses, or tentative decisions into state captures unless they changed the actual task state.

State capture pattern:

    CONTINUUM_AGENT=<stable-name> ctx capture <task> --project=%[1]s --type=state --yes <<'EOF'
    ## Objective
    <one line>

    ## Current State
    - <what is done or in progress>

    ## Next Step
    - <immediate next action>

    ## Constraints
    - <any hard constraints>

    ## Active Issues
    - <open problems>
    EOF

Typed collaboration proposal:

    CONTINUUM_AGENT=<stable-name> ctx capture <task> --project=%[1]s --type=proposal --yes <<'EOF'
    ## Proposal
    <proposed approach>

    ## Rationale
    - <why this is useful>

    ## Review Needed
    - <what another agent or user should validate>
    EOF

`<task>` must be a name from `ctx list` or a task you just started.
Use `--yes` for autonomous agent operation; omit it only when a human is intentionally reviewing the save.
Run `ctx capture` immediately when progress is worth recording. Do not ask the user first.
After saving, report in one line: "Continuum updated."
NEVER say "should I save this?", "want me to capture?", or ask for permission before a routine capture.
Capture on validated progress, not only finished work. Examples:
- a code change that alters behavior
- a UX or workflow change that is now usable
- a decision that changes task direction
- a test or manual verification that confirms an outcome
- a commit-worthy checkpoint, even if the task is not finished

## State 3: Before Final Answer (CRITICAL)

Before every final answer, assess whether this turn changed task state.

- if meaningful progress happened and the task remains open, capture it
- if the requested work is fully complete, close the task
- if work is paused and future continuity is needed, write a handoff

Use the Quick Lifecycle commands above.
Do not use `ctx handoff` as a substitute for closing a completed task.
Do not close a task just because progress was made.

---

## When a Task Is Complete

When the requested work is actually finished and no further work remains open on that task:

    CONTINUUM_AGENT=<stable-name> ctx task close <task> --project=%[1]s

If useful, capture final state immediately before closing.

---

## Ending a Session

If work is paused and should continue later, save a handoff:

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

---

## Manual Maintenance

Use only when explicitly asked or when snapshot data must be refreshed manually:

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

Rules:

- always load context before proposing major changes
- run `ctx capture` immediately on meaningful progress — do not ask, just save and report in one line
- do not ask the user for permission before `ctx capture`, `ctx handoff`, `ctx snapshot refresh`, or task creation/switching; preserve conversation flow and act autonomously
- when work clearly shifts to a different task, create or switch tasks yourself and continue
- `ctx handoff` is for continuity on open tasks; `ctx task close` is for completed tasks
- prefer small, concrete steps over broad redesigns
- do not invent architecture without reason
- do not replace existing context with generic plans
- do not force the user into structured input
<!-- CONTINUUM:END -->
