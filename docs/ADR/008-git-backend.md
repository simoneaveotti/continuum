# ADR 008: Git backend for Continuum storage

## Status

Accepted

## Context

Continuum persists all agent data under `~/.ctx/`, but the platform must stay local-first, append-only, and privacy-aware. Multiple agents may work on the same task concurrently and different machines must eventually see the same history. Prior discussions outlined an append-only storage model plus discrete commands; we need an architecture decision that makes git an internal transport without surfacing git to the user.

## Decision

- Treat `~/.ctx/` itself as a hidden git repository. The repository stays append-only: every `ctx capture`, `ctx handoff`, and `ctx snapshot refresh` writes a timestamped file (`snapshot.<RFC3339>.<uuid-6>.md`, `handoff.<...>.md`), commits that file, and pushes to `origin/main` in a best-effort fashion.
- `ctx init` bootstraps git plus `.gitignore`, seeds an initial commit, and optionally clones a remote via `--remote=<url>` when the storage directory is empty. `ctx init <project>` commits the new project file as `init(<project>): project initialized`.
- `ctx sync [--remote=<url>]` orchestrates the read/write boundary: it adds the remote if requested, bootstraps `origin/main` with the first push when the remote is empty, otherwise runs `git pull --rebase origin main` followed by `git push origin main`, clears a `local/unsynced` marker on success, logs failures in `local/git.log` (rotated at ~1 MiB), and prints the latest log line for visibility. Failed pushes append the HEAD hash to `local/unsynced`.
- `ctx context` always attempts a pull before loading state; it warns (non-blocking) if the pull fails and reports the number of unsynced commits so agents know whether their work is still local.
- Additional commands enforce recovery and cleanup:
  * `ctx repair` runs `git fsck`, aborts in-progress merge/rebase operations, and, on corruption, copies the whole `~/.ctx/` into a timestamped backup before instructing the user to clone the remote.
  * `ctx snapshot clean <task> [--keep=N]` retains only the newest `N` snapshots for a task, stages and commits the deletions, and never touches handoff files.
  * `ctx task delete <task>` / `ctx project delete <project>` remove the directory, stage the deletions, and commit them so git history reflects the change.
- The git command wrapper (`internal/vcs`) exposes only the narrow operations Continuum needs (`init`, `clone`, `commit`, `push`, `pull`, `fsck`, `abort`, plus helpers like `RevListCount`, `RemoteURL`, `HeadCommit`, `Execute`). Errors are wrapped and logged without leaking git stack traces.

## Consequences

- The append-only model remains the primary data model; git acts as transport/history. Users never run git themselves.
- Concurrent writes succeed even if the remote is unreachable; unsynced commits are recorded locally and sync can be retried manually via `ctx sync`.
- Agents have explicit commands for repair (`ctx repair`), cleanup (`ctx snapshot clean`), and deletion, keeping history audit-friendly.
- Observability is preserved through `local/unsynced`, `local/git.log`, and the `ctx sync` log tail, so operators can monitor degraded states without digging through git internals.

## Session Bootstrap

The current model now includes a storage-level `ctx resume` command as the opinionated user entrypoint for every new session before any project is chosen.

Expected semantics:

- operate at storage scope, not project scope
- validate that `~/.ctx/` is a usable git repository
- detect and repair in-progress merge/rebase state when safe
- stop with a clear diagnosis if the working tree is dirty
- run the equivalent of `ctx sync` when the repository is clean
- print a global orientation summary such as available projects, open tasks, and unsynced state

Expected user-facing failure handling:

- if the storage is already clean and healthy, `ctx resume` should succeed quietly and orient the user
- if a merge/rebase is in progress, `ctx resume` should repair it automatically when safe, or stop with a direct remediation message
- if the working tree is dirty, `ctx resume` should stop and tell the user to resolve or discard local changes before continuing
- if the remote is unavailable, `ctx resume` should say that work may continue locally but the session is not fully aligned

This keeps `ctx sync` as the transport primitive while giving users and agents a single command that answers: "is this storage ready for work?", plus what to do next when the answer is no.
