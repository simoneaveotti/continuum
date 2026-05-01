# Sync and Session Recovery

## Quick Start on a New Computer

If you already have Continuum data from another machine, the setup depends on how that storage is managed.

If your Continuum storage is synced to a git remote:

This assumes the remote repository already exists and is dedicated to Continuum storage.

```bash
# create a dedicated private git repository for Continuum storage first
ctx init --remote=git@github.com:you/continuum-state.git
git -C "${CONTINUUM_PATH:-$HOME/.ctx}" config user.name "Your Name"
git -C "${CONTINUUM_PATH:-$HOME/.ctx}" config user.email "you@example.com"
ctx resume
ctx context --project=my-project
```

If your Continuum storage exists only on the old machine:

```bash
# copy ${CONTINUUM_PATH:-$HOME/.ctx} from the old machine to the new one
git -C "${CONTINUUM_PATH:-$HOME/.ctx}" config user.name "Your Name"
git -C "${CONTINUUM_PATH:-$HOME/.ctx}" config user.email "you@example.com"
ctx resume
ctx context --project=my-project
```

Notes:

- `ctx init` uses `~/.ctx/` by default
- set `CONTINUUM_PATH` if this machine should use a different storage location
- use a dedicated git repository for Continuum storage, not your application repository
- if you use different git identities for different storage instances, prefer repo-local config with `git -C "${CONTINUUM_PATH:-$HOME/.ctx}" config ...` instead of `--global`
- git-backed storage requires `git user.name` and `git user.email` before `ctx resume`, `ctx capture`, or `ctx sync`
- `ctx resume` should be the first command at the start of a session on an existing storage
- `ctx sync` is still valid when you only want to pull/push storage state; `ctx resume` wraps health checks, safe repair, sync, and global orientation
- agents inside a known project should start with `ctx context --project=<project>` and `ctx list --project=<project>` rather than `ctx resume`
- if the old storage looks corrupted or incomplete, back it up first and run `ctx repair` before relying on it

## Git Sync & Recovery

Continuum keeps `~/.ctx/` inside a git repo. Use `ctx sync` to pull/push with a remote, and `ctx repair` when git detects corruption.

Before using git-backed writes, ensure git identity is configured for Continuum commits.
If you use different identities for different Continuum storage instances, prefer repo-local configuration on the active Continuum storage path:

```bash
git -C "${CONTINUUM_PATH:-$HOME/.ctx}" config user.name "Your Name"
git -C "${CONTINUUM_PATH:-$HOME/.ctx}" config user.email "you@example.com"
```

Use `--global` only if this machine should use the same git identity for every Continuum storage instance.

If your machine hostname is unstable or polluted by local aliases such as DDEV entries in `/etc/hosts`, set a stable local Continuum host identity:

```bash
ctx config set host workstation-rome
```

For temporary overrides in CI or scripted environments, `CONTINUUM_HOST` takes precedence over the saved local value.

- `ctx sync [--remote=<url>]` runs `git pull --rebase origin main` and `git push origin main`, adds the remote when `--remote` is provided, and bootstraps `origin/main` on the first sync if the remote is still empty
- failed pushes mark the commit hash in `~/.ctx/local/unsynced`
- `ctx repair` runs `git fsck`, aborts in-progress merge/rebase state, and can create a backup before suggesting a remote restore path
- `ctx repair --activity` deduplicates and normalizes `events/activity.ndjson` when the shared activity log accumulates duplicate or malformed lines
- `ctx init --remote=<url>` clones a fresh Continuum repo when `~/.ctx/` is empty

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

## Watch Mode

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

## History

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
