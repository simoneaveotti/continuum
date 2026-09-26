# ADR 003: Project Identification

## Status

Superseded — retained as historical design context. The current CLI detects a
project from the working-directory name or accepts `--project=<name>` after a
command; it does not support `CONTINUUM_PROJECT` or global flags.

## Context

Commands need to know which project context to use. The user should not think about this.

## Decision

Project determined in order:

1. **CONTINUUM_PROJECT** env var
2. **--project** flag (explicit)
3. **Directory name** (fallback)

```bash
# Via environment (recommended for agents)
export CONTINUUM_PROJECT=my-project

# Explicit
ctx --project=my-project context

# Fallback (directory name)
cd /path/to/my-project
ctx context
```

## Principle

The system figures it out. The user doesn't think about it.

Only needed when:
- Setting up the environment
- Manual maintenance

## Storage

Project metadata stored in:
```
.ctx/projects/<project-name>/
├── project.md    # project context
└── tasks/        # task directory
```
