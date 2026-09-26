# ADR 004: Init Semantics

## Status

Superseded — retained as historical design context. Current setup is `ctx init`
followed by `ctx project init <project>`; `ctx init <project>` is unsupported.

## Context

Setting up continuum for the first time. Should be minimal - user does it once.

## Decision

Two modes:

```bash
# Session only (first time setup)
ctx init

# Session + Project
ctx init <project-name>
```

### Session Init

Creates global context:
```
.ctx/
├── profile.md          # user preferences
├── notes.md            # user notes (optional)
├── skills/
│   └── agent.md        # agent instructions
├── local/
└── exports/
```

### Project Init

Creates project context:
```
.ctx/projects/<project>/
├── project.md          # project context
└── tasks/              # task directory
```

## When to Use

- **Once per machine**: `ctx init`
- **When starting project**: `ctx init <project>`
- **Not during work**: conversation handles context

## Principle

Setup once. Then just work.
