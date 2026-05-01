# ADR 005: Command Structure

## Status

Proposed

## Context

CLI must be minimal. The user interacts conversationally, not via CLI.

## Decision

Minimal commands:

```bash
# Setup (once)
ctx init              # session
ctx init <project>   # session + project

# Primary (conversational)
ctx capture           # distill context from conversation
ctx context           # deliver context to agent

# Maintenance (optional)
ctx profile           # edit user profile
ctx project           # edit project context
ctx tasks             # list tasks

# Sync
ctx sync              # sync with cloud
ctx export <task>     # export to file
ctx import <file>     # import from file
```

## Core: ctx capture

The key command:

```
$ ctx capture

Agent: "Current state:
- Snapshot is working
- Parsing has issues
- Decision: local-first
- Next: fix cleanPrefill

Apply? [y]es [e]dit [n]o"
```

- Agent proposes summary (natural language)
- System maps to structured fields
- User confirms (y/e/n)

## Core: ctx context

Delivers operational context to agent:

```
$ ctx context

# USER
[profile preferences]

# PROJECT
[project context]

# CURRENT STATE
[snapshot]

# TRANSITION
[handoff]
```

Compact. Operational. 500-1000 tokens.

## Principles

1. **Rare use**: most interaction is conversation, not CLI
2. **Capture is key**: only command that changes state
3. **Context is read**: delivers to agents
4. **Simple**: fewer commands, clearer purpose

## Hidden Layer

System maintains:
- snapshot.md
- handoff.md
- project.md
- profile.md

These are derived from capture, not written manually.
