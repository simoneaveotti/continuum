# Continuum

A terminal-first context orchestration tool for AI-assisted software development.

Continuum reduces the cost of restarting work across machines, sessions, projects, and agents. It is local-first, privacy-aware, agent-agnostic, and offline-capable.

## Quick Start

If you want to try Continuum local-only in under a minute:

```bash
ctx init
ctx project init my-project
ctx agent install --project=my-project
ctx context --project=my-project
```

Then, after meaningful progress:

```bash
ctx capture my-task --project=my-project --yes <<'EOF'
## Objective
One-line goal

## Current State
- What is done

## Next Step
- What should happen next

## Constraints
- Hard constraints

## Active Issues
- Open issues
EOF
```

## Quick Start on a New Computer

If you already have Continuum data from another machine, the setup depends on how that storage is managed.

If your Continuum storage is synced to a git remote:

This assumes the remote repository already exists and is dedicated to Continuum storage.

```bash
# create a dedicated private git repository for Continuum storage first
ctx init --remote=git@github.com:you/continuum-state.git
git -C "${CONTINUUM_PATH:-$HOME/.continuum}" config user.name "Your Name"
git -C "${CONTINUUM_PATH:-$HOME/.continuum}" config user.email "you@example.com"
ctx resume
ctx context --project=my-project
```

If your Continuum storage exists only on the old machine:

```bash
# copy ${CONTINUUM_PATH:-$HOME/.continuum} from the old machine to the new one
git -C "${CONTINUUM_PATH:-$HOME/.continuum}" config user.name "Your Name"
git -C "${CONTINUUM_PATH:-$HOME/.continuum}" config user.email "you@example.com"
ctx resume
ctx context --project=my-project
```

Notes:

- `ctx init` uses `~/.continuum/` by default
- set `CONTINUUM_PATH` if this machine should use a different storage location
- use a dedicated git repository for Continuum storage, not your application repository
- if you use different git identities for different storage instances, prefer repo-local config with `git -C "${CONTINUUM_PATH:-$HOME/.continuum}" config ...` instead of `--global`
- git-backed storage requires `git user.name` and `git user.email` before `ctx resume`, `ctx capture`, or `ctx sync`
- `ctx resume` should be the first command at the start of a session on an existing storage
- `ctx sync` is still valid when you only want to pull/push storage state; `ctx resume` wraps health checks, safe repair, sync, and global orientation
- agents inside a known project should start with `ctx context --project=<project>` and `ctx list --project=<project>` rather than `ctx resume`
- if the old storage looks corrupted or incomplete, back it up first and run `ctx repair` before relying on it

## What It Does

Continuum gives you an explicit way to:

- preserve working context outside the repo
- resume work across sessions, machines, and agents
- keep the user in control of what gets persisted

It does not replace your workflow. It supports it.

## Core Model

Continuum separates three concerns:

- Environment: managed locally with `ctx init`
- Agent behavior: injected into the workspace with `ctx agent install`
- Context/state: stored outside the repo and retrieved via `ctx`

There is no automatic discovery. Agents must be told to use Continuum explicitly.

## Important Constraint

Continuum stores state outside the workspace (default: `~/.continuum/`).

- agents should not read or write `~/.continuum/` directly
- agents should retrieve context with `ctx context`
- agents should persist updates with `ctx capture`
- `ctx` is the only supported bridge between the workspace and Continuum storage

## Installation

```bash
go build -o ctx ./cmd/
sudo cp ctx /usr/local/bin/
xattr -c /usr/local/bin/ctx 2>/dev/null || true
```

### Distribution

Continuum is a single statically-linked Go binary (`ctx`) with no runtime dependency beyond `git` and optional `gpg` for encrypted exports.

Primary release targets:

| Platform | Arch | Notes |
|---|---|---|
| macOS | arm64 | Apple Silicon primary target |
| macOS | amd64 | Intel Mac |
| Linux | amd64 | x86_64 servers and desktops |
| Linux | arm64 | Raspberry Pi and ARM servers |
| Windows | amd64 | Secondary target |

Release automation is handled by GitHub Actions and GoReleaser.

- push a `v*` tag to trigger `.github/workflows/release.yml`
- GoReleaser builds archives, checksums, GitHub draft releases, and a Homebrew formula update
- the direct install fallback is `install.sh`

Typical release flow:

```bash
go test ./...
git tag v0.6.0 -m "v0.6.0"
git push origin v0.6.0
```

Homebrew target:

```bash
brew tap simoneaveotti/continuum
brew install continuum
```

Direct install fallback:

```bash
curl -fsSL https://raw.githubusercontent.com/simoneaveotti/continuum/main/install.sh | bash
```

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

### 3. Load Context

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

`--compact` is the recommended agent bootstrap path. It keeps the same authoritative state but omits empty fields and compresses labels such as `PROJECT`, `CURRENT STATE`, and `NEXT STEP` into short keys like `PRJ`, `STATE`, and `NEXT`. The full context remains available on demand.

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

## Next Step
- <immediate next action>

## Constraints
- <hard constraints>

## Active Issues
- <open issues>
EOF
```

The same pattern applies to `ctx handoff` and `ctx snapshot refresh`.

## Commands

### Setup

- `ctx init`
- `ctx init --remote=<url>`
- `ctx project list`
- `ctx project init <project>`
- `ctx project onboard <project>`
- `ctx project delete <project>`
- `ctx config set host <name>`
- `ctx agent install --project=<name>`
- `ctx agent remove [--project=<name>]`

### Context

- `ctx resume`
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
- `ctx repair`
- `ctx sync [--remote=<url>] [--prefer=local|remote] [--force]`
- `ctx watch [--project=<name>] [--interval=<duration>]`

### Task Management

- `ctx list`
- `ctx list --all`
- `ctx list --status=<active|closed>`
- `ctx task start <task> --project=<name>`
- `ctx task close <task> --project=<name>`
- `ctx task reopen <task> --project=<name>`
- `ctx task delete <task> --project=<name>`
- `ctx handoff <task> --project=<name>`
- `ctx handoff <task> --project=<name> --yes`
- `ctx snapshot refresh <task> --project=<name>`
- `ctx snapshot refresh <task> --project=<name> --yes`
- `ctx snapshot clean <task> --project=<name> [--keep=N]`

### Import / Export

- `ctx export <task>`
- `ctx export --project=<name1,name2>`
- `ctx export --session`
- `ctx import <path>`

`ctx export` now produces archives that `ctx import` can restore directly.

## Advanced

### Git Sync & Recovery

Continuum keeps `~/.continuum/` inside a git repo. Use `ctx sync` to pull/push with a remote, and `ctx repair` when git detects corruption.

Before using git-backed writes, ensure git identity is configured for Continuum commits.
If you use different identities for different Continuum storage instances, prefer repo-local configuration on the active Continuum storage path:

```bash
git -C "${CONTINUUM_PATH:-$HOME/.continuum}" config user.name "Your Name"
git -C "${CONTINUUM_PATH:-$HOME/.continuum}" config user.email "you@example.com"
```

Use `--global` only if this machine should use the same git identity for every Continuum storage instance.

If your machine hostname is unstable or polluted by local aliases such as DDEV entries in `/etc/hosts`, set a stable local Continuum host identity:

```bash
ctx config set host workstation-rome
```

For temporary overrides in CI or scripted environments, `CONTINUUM_HOST` takes precedence over the saved local value.

- `ctx sync [--remote=<url>]` runs `git pull --rebase origin main` and `git push origin main`, adds the remote when `--remote` is provided, and bootstraps `origin/main` on the first sync if the remote is still empty
- failed pushes mark the commit hash in `~/.continuum/local/unsynced`
- `ctx repair` runs `git fsck`, aborts in-progress merge/rebase state, and can create a backup before suggesting a remote restore path
- `ctx repair --activity` deduplicates and normalizes `events/activity.ndjson` when the shared activity log accumulates duplicate or malformed lines
- `ctx init --remote=<url>` clones a fresh Continuum repo when `~/.continuum/` is empty

Recommended start-of-session flow:

```bash
ctx resume
ctx context --project=<project> --compact
```

Interpretation:

- start every new work session with `ctx resume` before choosing a project
- `ctx resume` validates storage health, attempts repair when safe, runs sync when the repo is clean, and prints a global orientation view before any project is selected
- after `ctx resume`, use `ctx context` or `ctx list` to choose what to resume
- use `ctx sync` directly when you only need to synchronize storage and do not need repair/orientation output
- use `ctx context --project=<project> --compact` for project/task operational state; it is not a replacement for storage-level `ctx resume`
- use full `ctx context --project=<project>` when compact output is insufficient for the requested work

Why this matters:

- if you skip the start-of-session sync step, you can begin work on stale local state
- later writes may still succeed locally, but push failures and unsynced commits will surface only after you are already in the middle of work
- project/task selection becomes less trustworthy because you may be looking at an outdated local snapshot

- if storage is healthy and clean: `ctx resume` syncs and shows you what is available to resume
- if a merge/rebase is stuck: `ctx resume` should repair it automatically when safe, or stop with a clear message
- if the working tree is dirty: `ctx resume` should stop and tell you to resolve local changes before continuing
- if the remote is missing or unreachable: `ctx resume` should say so explicitly and leave the storage in local-only mode until sync works again

Typical flows:

```bash
ctx sync --remote=<url>
ctx init --remote=<url>
```

The `local/unsynced` file lists pending commit hashes, and `local/git.log` records real git errors. `ctx context` surfaces pending unsynced commits so agents can act accordingly.

### Watch Mode

`ctx watch [--project=<name>] [--interval=<duration>]` is the current visibility MVP.

- it polls a shared append-only activity stream in Continuum storage
- if `--project` is omitted, it watches every project
- it prints lifecycle events such as task creation, task writes, export/import, repair, agent install/remove, and sync activity
- `ctx watch --tui` opens a terminal UI on top of the same stream
- inside the TUI, `p` cycles the project filter and `P` returns to all projects

Example:

```bash
ctx watch
ctx watch --project=continuum
ctx watch --project=continuum --interval=5s
ctx watch --tui
```

### History

`ctx history [--project=<name>] [--task=<name>] [--limit=<n>] [--since=<duration>]` prints a static narrative timeline from the Continuum activity stream.

- it is meant for retrospective review, unlike `ctx watch`, which is live
- it renders events chronologically, oldest to newest
- it can be scoped to a project or a specific task
- it can be trimmed to the most recent N events or a recent time window

Example:

```bash
ctx history --project=continuum
ctx history --project=continuum --task=history-command
ctx history --project=continuum --limit=20 --since=7d
```

### Export Encryption

Continuum can protect exported archives with a passphrase via `ctx export --encrypt`.

- the default format is `aes-gcm-v2`, which uses Argon2id-derived keys and embeds the parameters needed for decryption
- this is practical file protection, not enterprise key management
- passphrases are read from stdin; interactive entry is safer than piping literals from shell commands

### Storage Patterns

Continuum stores all state in a local directory (default: `~/.continuum/`).
`CONTINUUM_PATH` overrides this location.

Because each `CONTINUUM_PATH` is an independent storage instance with its own git repository, remote, and project set, you can run multiple isolated Continuum instances side by side with different sync policies and access rules.

#### Pattern 1 — Single storage, no remote

Everything is local. No sync. No cloud dependency.

```bash
ctx init
ctx project init my-project
ctx context --project=my-project
```

Use when: single machine, personal projects, getting started.

#### Pattern 2 — Single storage with remote

One storage, synced across machines through a git remote.

Create a dedicated private git repository for this storage first, for example on GitHub, GitLab, or Gitea.
Do not reuse your application repository for Continuum state.

```bash
ctx sync --remote=git@github.com:you/continuum-state.git
```

From a second machine:

This assumes the same remote repository already exists and is reachable from that machine.

```bash
ctx init --remote=git@github.com:you/continuum-state.git
```

Use when: multiple workstations, same context everywhere.

#### Pattern 3 — Multiple storage instances for sensitive projects

Run a second Continuum instance in a different directory:

```bash
# Normal projects — synced
ctx context --project=my-app

# Sensitive project — local only
CONTINUUM_PATH=~/.continuum-private ctx context --project=sensitive-project
```

The sensitive storage has no remote configured. Agents working there must be bootstrapped with the correct `CONTINUUM_PATH`.

Use when: regulated, healthcare, client-confidential, or otherwise non-syncable work.

#### Pattern 4 — Multiple storage instances with separate remotes

Each storage instance can point to a different remote:

```text
~/.continuum/          -> github.com/you/ctx-work
~/.continuum-private/  -> gitea.yourserver.com/you/ctx-private
~/.continuum-client/   -> github.com/you/ctx-client-x
```

Each instance stays independent: different projects, different remotes, different sync rules.

Use when: multi-client consulting, residency separation, or strict project boundaries.

#### Pattern 5 — Work / personal separation

Common setup for employed developers or consultants:

```text
~/.continuum/          -> company remote
~/.continuum-personal/ -> personal remote
```

You can switch automatically with project-specific shell config such as:

```bash
# in .envrc for personal projects
export CONTINUUM_PATH=~/.continuum-personal
```

Use when: company and personal work must stay fully separate.

#### Security Model

Continuum does not enforce access control between agents. Separation here is path isolation: an agent that does not know a `CONTINUUM_PATH` value cannot access that storage instance.

This is path-level isolation, not a cryptographic guarantee. For stronger local guarantees, keep sensitive storage unsynced and lock down directory permissions, for example:

```bash
chmod 700 ~/.continuum-private
```

Agents working on sensitive storage should be bootstrapped with the correct `CONTINUUM_PATH` explicitly set in their environment or launcher.

## Storage Model

Continuum stores all data locally (default: `~/.continuum/`).

Example for project `my-project` and task `my-task`:

```text
~/.continuum/
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
├── local/
│   ├── identity.json
│   ├── git.log
│   └── unsynced
└── exports/
```

Edit `~/.continuum/agent-targets.txt` to add support for any agent instruction file (`AGENTS.md`, `GEMINI.md`, `.cursorrules`, and so on).

## Design Principles

- Local-first
- Privacy-aware
- Agent-agnostic
- Offline-capable
- Explicit, not automatic
- Minimal and composable

## Built with Continuum

Continuum was built using itself — across multiple machines, five different
coding agents, and dozens of sessions.

The full development history is available as an importable snapshot:

```bash
ctx import https://github.com/simoneaveotti/continuum/releases/latest/download/continuum-history.zip
ctx history --project=continuum
```

## Summary

Continuum does not manage your work.

It preserves continuity.

The agent does not guess context.
It retrieves it with `ctx`.

The user does not rewrite context.
Continuum carries it across sessions.
