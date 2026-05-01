# ADR 001: Continuum Location

## Status

Proposed

## Context

Continuum is the "memory" that preserves context between human-agent conversations. It must be:
- Accessible from any machine
- Syncable across devices
- Independent from project repositories
- Available to agents when they need to resume work

The user interacts naturally. The system stores silently.

## Decision

`.ctx` is stored in a **global/shared location**, outside any project directory.

```
~/.ctx/              # or CONTINUUM_PATH env var
├── profile.md            # global user profile
├── notes.md              # global notes
├── skills/
│   └── agent.md          # agent skill
├── projects/
│   └── <project>/
│       ├── project.md    # project-specific
│       └── tasks/
│           └── <task>/
│               ├── snapshot.md
│               ├── handoff.md
│               ├── notes.md
│               └── discussion.md
├── local/
└── exports/
```

## Consequences

### Pros
- Works with any project location (cloned repos, different machines)
- No need to write in project directories (read-only repos)
- Single source of truth for all context
- Easy to share with agents (one location)

### Cons
- Requires environment setup (CONTINUUM_PATH)
- Must remember which projects exist in continuum
- More complex path handling in code
