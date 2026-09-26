# Continuum

A terminal-first context orchestration tool for AI-assisted software development.

## What It Does

Continuum gives you an explicit way to:

- preserve working context outside the repo
- resume work across sessions, machines, and agents
- keep the user in control of what gets persisted

It does not replace your workflow. It supports it.

The user initializes storage, creates projects, and installs bootstrap
instructions into the workspace. When an agent starts, it reads those
instructions and knows to retrieve context via `ctx` before doing anything else.
How much the agent does from there depends on the agent and how it was configured.

## Core Model

Continuum separates three concerns:

- **Environment** — managed by the user (`ctx init`, `ctx project init`)
- **Agent behavior** — injected into the workspace (`ctx agent install`)
- **Context/state** — stored outside the repo, retrieved via `ctx`

There is no automatic discovery. The agent must be explicitly bootstrapped.

Three commands worth distinguishing:

    ctx resume    — for the user: validates storage health, syncs, and orients
    ctx sync      — for the user: pull/push storage state without health checks
    ctx context   — for the agent: loads operational state for a specific project

## Important Constraint

Continuum stores state outside the workspace (default: `~/.ctx/`).

- agents should not read or write `~/.ctx/` directly
- agents should retrieve context with `ctx context`
- agents should persist updates with `ctx capture`
- `ctx` is the only supported bridge between the workspace and Continuum storage

## Installation

### Install a release (recommended)

The installer downloads the latest release, installs `ctx` and its bundled
templates in `~/.local/bin`, and preserves executable permissions. On macOS it
also clears a possible quarantine attribute and applies an ad-hoc signature
when the system tools are available.

```bash
curl -fsSL https://raw.githubusercontent.com/simoneaveotti/continuum/main/install.sh | bash
```

Ensure `~/.local/bin` is on your `PATH` if the installer tells you it is not.
To inspect the script before running it:

```bash
curl -fsSL https://raw.githubusercontent.com/simoneaveotti/continuum/main/install.sh -o /tmp/continuum-install.sh
less /tmp/continuum-install.sh
bash /tmp/continuum-install.sh
```

For a system-wide installation, set the destination explicitly:

```bash
curl -fsSL https://raw.githubusercontent.com/simoneaveotti/continuum/main/install.sh | sudo env INSTALL_DIR=/usr/local/bin bash
```

### Build from source

Building from source requires Go 1.26.1 or later. This is primarily useful for
development; it does not require `sudo` when installed in the default user
location.

```bash
go build -o ctx ./cmd/
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
INSTALL_PREFIX="$(cd "$(dirname "$INSTALL_DIR")" && pwd)"
install -d "$INSTALL_DIR" "$INSTALL_PREFIX/share/continuum/templates"
install -m 755 ctx "$INSTALL_DIR/ctx"
cp -R templates/. "$INSTALL_PREFIX/share/continuum/templates/"
```

The template copy is required for an installation outside a source checkout.
No separate `chmod`, `xattr`, or macOS-specific copy step is required.

For release automation, Homebrew tap, and distribution details, see [docs/RELEASING.md](docs/RELEASING.md).

## Core Usage

### 1. Initialize

```bash
ctx init
ctx project init my-project
```

Useful variants:

```bash
ctx init --templates=./templates
ctx init --force
ctx project init my-project
ctx project onboard my-project
ctx config set host workstation-rome
```

### 2. Install Agent Bootstrap

```bash
ctx agent install --project=my-project
```

If the bootstrap templates changed and you want to re-inject them into existing files:

```bash
ctx init --force
ctx agent install --project=my-project --force
```

To check or refresh installed bootstrap instructions later:

```bash
ctx agent status --project=my-project
ctx agent update --project=my-project
```

If the current directory maps unambiguously to a Continuum project, `--project` may be omitted.
`ctx agent update` is idempotent: it only rewrites stale or unknown installed bootstrap blocks unless `--force` is passed.

### 3. Load Context

Before starting a new session on existing storage, run:

```bash
ctx resume
```

Use `ctx resume --verbose` when you need the per-project active-task breakdown.

When an agent starts, it should read the injected bootstrap instructions and load context through `ctx`:

```bash
ctx context --project=my-project --compact
```

Use:

- `ctx list --project=my-project` to inspect active tasks
- `ctx list --project=my-project --all` to include closed tasks
- `ctx context <task> --project=my-project --compact` to focus on one task with the token-efficient digest
- `ctx context <task> --project=my-project` when the compact digest is not enough and you need the full context
- `ctx task start my-task --project=my-project` to create a task inside that project

`--compact` is the default for agent bootstrap. Use full context only when needed.

### 4. Persist Progress

The three write patterns are:

- `ctx capture <task> --project=<name>`: save current task state
- `ctx handoff <task> --project=<name>`: leave a handoff for another session or agent
- `ctx snapshot refresh <task> --project=<name>`: manually rewrite a fuller structured snapshot

The most common path is `ctx capture`:

```bash
ctx capture my-task --project=my-project <<'EOF'
## Objective
<one line>

## Current State
- <what is done or in progress>

## Decisions (Locked)
- <why choices were made>

## Next Step
- <immediate next action>

## Constraints
- <hard constraints>

## Active Issues
- <open issues>
EOF
```

`ctx capture` defaults to `--type=state`, which writes `snapshot.*.md` and drives `ctx context`.
For multi-agent collaboration notes that should not overwrite task state, use a typed capture:

```bash
ctx capture my-task --project=my-project --type=proposal --yes <<'EOF'
## Proposal
Use --type for collaboration artifacts.

## Rationale
- Keeps one command while separating proposals from state.
EOF
```

Supported capture types are `state`, `proposal`, `request`, `response`, and `decision`.
`ctx context` keeps state from the latest snapshot and adds compact proposal/request/response/decision summaries when present.
Use `ctx artifact list <task> --project=<name>` and `ctx artifact show <task> <filename> --project=<name>` to inspect collaboration artifacts directly.
Use `ctx resolve <task> <filename> --project=<name>` to move an artifact out of open proposal/request counts without deleting it.
Use `ctx capture <task> --project=<name> --type=decision --resolves=<filename> --yes` to save a decision and resolve a linked proposal/request in one step.

For non-interactive environments where approval was already given, add `--yes`:

```bash
ctx capture my-task --project=my-project --yes <<'EOF'
## Objective
<one line>

## Current State
- <what is done or in progress>

## Decisions (Locked)
- <why choices were made>

## Next Step
- <immediate next action>

## Constraints
- <hard constraints>

## Active Issues
- <open issues>
EOF
```

The same pattern applies to `ctx handoff` and `ctx snapshot refresh`.

For git sync, watch mode, history, and session recovery, see [docs/sync.md](docs/sync.md).
For storage patterns and multi-instance setups, see [docs/storage.md](docs/storage.md).
For export encryption, see [docs/encryption.md](docs/encryption.md).

## Commands

### Setup

- `ctx init`
- `ctx init --remote=<url>`
- `ctx project list`
- `ctx project init <project>`
- `ctx project onboard <project>`
- `ctx project delete <project> [--yes]`
- `ctx config set host <name>`
- `ctx agent install --project=<name>`
- `ctx agent status [--project=<name>]`
- `ctx agent update [--project=<name>] [--force]`
- `ctx agent remove`

### Context

- `ctx resume [--verbose]`
- `ctx context --project=<name>`
- `ctx context <task> --project=<name>`
- `ctx capture <task> --project=<name>`
- `ctx capture <task> --project=<name> --type=state|proposal|request|response|decision`
- `ctx capture <task> --project=<name> --type=decision --resolves=<filename>`
- `ctx artifact list <task> --project=<name> [--type=proposal|request|response|decision|all]`
- `ctx artifact show <task> <filename> --project=<name>`
- `ctx resolve <task> <filename> --project=<name>`
- `ctx capture <task> --project=<name> --yes`
- `ctx history [--project=<name>] [--task=<name>] [--limit=<n>] [--since=<duration>]`
- `ctx timeline [--project=<name>] [--task=<name>] [--limit=<n>] [--since=<duration>]`
- `ctx repair`
- `ctx sync [--remote=<url>] [--prefer=local|remote] [--force]`
- `ctx watch [--project=<name>] [--interval=<duration>] [--tui]`

### Task Management

- `ctx list`
- `ctx list --all`
- `ctx list --status=<active|closed>`
- `ctx task start <task> --project=<name>`
- `ctx task close <task> --project=<name>`
- `ctx task reopen <task> --project=<name>`
- `ctx task delete <task> --project=<name> [--yes]`
- `ctx handoff <task> --project=<name>`
- `ctx handoff <task> --project=<name> --yes`
- `ctx snapshot refresh <task> --project=<name>`
- `ctx snapshot refresh <task> --project=<name> --yes`
- `ctx snapshot clean <task> --project=<name> [--keep=N]`

### Skills

- `ctx skill list`
- `ctx skill show <name>`
- `ctx skill save <name> [--description=<text>] [--yes]`
- `ctx skill delete <name> [--yes]`

### Help

- `ctx --help` for common commands
- `ctx --help --verbose` for the full command, option, and example reference

### Import / Export

- `ctx export <task>`
- `ctx export --project=<name1,name2>`
- `ctx export --session`
- `ctx import <path>`

`ctx export` now produces archives that `ctx import` can restore directly.

## Storage Model

```text
~/.ctx/
├── profile.md
├── agent-targets.txt
├── skills/
│   └── agent.md
├── projects/
│   └── my-project/
│       ├── project.md
│       └── tasks/
│           └── my-task/
│               ├── snapshot.20260102T150405Z.a3f2c1.md
│               ├── snapshot.20260102T150432Z.b7c1e4.md
│               └── handoff.20260102T150000Z.c5d2f1.md
├── templates/
│   ├── profile.md
│   ├── project.md
│   ├── bootstrap.md
│   ├── agent.md
│   └── agent-targets.txt
├── events/
│   └── activity.ndjson
├── local/
│   ├── identity.json
│   ├── git.log
│   └── unsynced
└── exports/
```

## Design Principles

- Local-first
- Privacy-aware
- Agent-agnostic
- Offline-capable
- Explicit, not automatic
- Minimal and composable

## Built with Continuum

Continuum was built using itself — across multiple machines, multiple coding
agents, nine real projects, and dozens of sessions.

## Summary

Continuum does not manage your work.

It preserves continuity.

The agent does not guess context.
It retrieves it with `ctx`.

The user does not rewrite context.
Continuum carries it across sessions.
