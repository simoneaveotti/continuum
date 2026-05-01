# ADR 007: ctx context Output Specification

## Status

Accepted

## Goal

`ctx context` must output a compact, operational summary that allows a coding agent to resume work without reading raw internal Continuum files.

It must not dump markdown files verbatim.

It must distill them.

---

## Output Principles

The output must be:

- concise
- operational
- action-oriented
- readable by both humans and agents
- free of internal Continuum formatting noise
- free of repeated headings
- free of empty placeholder sections when possible

The output must help answer:

- what project is this?
- what are the important working rules?
- what are we currently working on?
- where did we stop?
- what should happen next?

---

## Output Structure

The output of:

    ctx context --project=<project>

should follow this structure.

### 1. Project Header

Required.

Example:

    PROJECT: travel-manager

### 2. Working Style

Optional but recommended.

Only include compact profile-level working preferences and rules.

Example:

    WORKING STYLE:
    - concise operational summaries
    - avoid re-explaining known context
    - preserve decisions
    - distinguish facts vs assumptions

### 3. Current Focus

Required.

This tells the agent what the likely active task is.

Example (single task):

    CURRENT FOCUS: test-task-2

Example (multiple tasks):

    CURRENT FOCUS: not yet defined (available: test-task-1, test-task-2)

### 4. Objective

Required if available.

Example:

    OBJECTIVE:
    Improve GPS sync reliability between frontend and backend

If unknown:

    OBJECTIVE:
    not yet defined

### 5. Current State

Required if available.

This is the main operational state of the current task.

Example:

    CURRENT STATE:
    - API endpoint exists
    - frontend partially implemented
    - GPS updates are unreliable

### 6. Locked Decisions

Optional.

Example:

    LOCKED DECISIONS:
    - local-first storage
    - do not rely on Git for Continuum state

### 7. Last Session

Optional but highly useful.

Example:

    LAST SESSION:
    - what was done: implemented initial retry loop
    - current state at stop: retry logic added but not validated end-to-end

### 8. Next Step

Required if available.

This is the most important action-oriented section.

Example:

    NEXT STEP:
    Validate retry logic in the frontend and confirm backend response handling

### 9. Open Issues

Optional.

Example:

    OPEN ISSUES:
    - intermittent position update failures
    - unclear timeout behavior

---

## What Must Be Removed

- raw section markers like `# USER`
- raw document titles like `PROFILE`, `PROJECT`, `TASK SNAPSHOT`
- repeated markdown headings
- placeholder values such as `...`
- empty bullet lines
- internal file-oriented labels

---

## Distillation Rules

### Profile
Extract only:
- working preferences
- rules

### Project
Extract only:
- project name
- meaningful summary
- constraints

### Task Snapshot
Extract:
- objective
- current state
- locked decisions
- next step
- active issues

### Task Handoff
Extract:
- what was done
- current state at stop

---

## Focus Selection Rules

### Project-Level Context (`ctx context --project=<project>`)

- If there is exactly one task with available context, treat it as the implicit active task
- In that case, promote its `OBJECTIVE`, `CURRENT STATE`, and `NEXT STEP` directly into the project-level output
- If there are multiple tasks, do not invent a current focus; show available task names instead

### Task-Level Context (`ctx context <task> --project=<project>`)

- Must read directly from the requested task snapshot/handoff
- Must produce the same operational truth as the project-level view for that task
- Must not degrade to `not yet defined` if the requested task already has snapshot data

The same task should not appear "defined" in one context view and "undefined" in another.

---

## Example Output

    PROJECT: travel-manager

    WORKING STYLE:
    - concise operational summaries
    - avoid re-explaining known context
    - preserve decisions
    - distinguish facts vs assumptions

    CURRENT FOCUS:
    - task: test-task-2

    OBJECTIVE:
    Improve GPS sync reliability between frontend and backend

    CURRENT STATE:
    - API endpoint exists
    - frontend partially implemented

    LOCKED DECISIONS:
    - local-first Continuum storage

    LAST SESSION:
    - what was done: implemented initial retry loop

    NEXT STEP:
    Validate retry logic in the frontend

    OPEN ISSUES:
    - intermittent position update failures
