# Continuum — Development Progress

## Version
Version: v0.6
Last updated: 2026-04-15

## Git Backend — Phase 3 (wire VCS into runtime) — COMPLETED
- `ctx init` → `git init -b main ~/.continuum/`, `.gitignore`, and initial commit
- Write paths (`ctx capture`, `ctx handoff`, `ctx snapshot refresh`) now run `git add + commit + push` after each atomic write (push is best-effort / degraded)
- Read path (`ctx context`) issues a `git pull --rebase origin main --quiet` before loading and warns/logs when `local/unsynced` is non-empty
- `local/unsynced` marker is updated on push failure and cleared on successful sync; `ctx sync` prints the last `local/git.log` entry

## Git Backend — Phase 4 (new commands) — COMPLETED
- `ctx init --remote=<url>` clones a fresh Continuum repo when `~/.continuum/` is empty and fails fast on pre-existing data
- `ctx sync [--remote=<url>] [--prefer=local|remote] [--force]` explicitly pulls/pushes, adds the remote if requested, bootstraps `origin/main` when the remote is empty, and can resolve dirty local storage according to an explicit preference
- Dirty local Continuum storage is now detected before pull/rebase; sync reports the local cause directly instead of surfacing generic remote failures
- `ctx repair` runs `git fsck`, aborts in-progress operations, and copies `~/.continuum/` to `~/.continuum-backup-<timestamp>/` when corruption is detected
- `ctx repair --activity` deduplicates malformed/duplicate activity log entries and reorders valid entries by timestamp
- `ctx snapshot clean <task> [--keep=N]` prunes older `snapshot.*.md` files, stages the removals, and commits the cleanup
- `ctx task delete <task>` and `ctx project delete <project>` remove entire directories, stage them via git, and commit so the history stays consistent

### Other pending
- `ctx capture` edit mode — implement via `$EDITOR` env var
- Encryption follow-up — consider tuning Argon2 parameters further based on real-machine measurements; current `aes-gcm-v2` uses Argon2id with embedded parameters.
- `ctx summarize <task>` — planned context compaction for long-running tasks; not implemented yet.

---

## Product Notes

- Visibility UX is currently too opaque for users: agents can update Continuum state, but users must already know to run `ctx context` manually to inspect it
- `CONTINUUM_PATH` already enables multiple independent Continuum storage instances side by side, which supports work/personal separation, per-client separation, or fully local-only sensitive storage without new runtime changes
- Distribution can stay simple: one static `ctx` binary, GitHub Releases via GoReleaser, optional Homebrew tap, and a direct `install.sh` fallback
- `ctx watch` is now the visibility MVP:
  - streams readable lines when new snapshots or handoffs appear
  - uses a shared append-only activity stream
  - lower complexity than a full TUI
  - high immediate value for users supervising agent work
- `ctx watch --tui` is available as the operator view for activity monitoring, including interactive project filtering
- Diff-style capture output is a strong companion feature: show what changed relative to the previous saved state, not just the new full state
- Optional review mode (`--review`) is worth exploring for users who want propose-and-approve flow without giving up default automation

---

## Recent Changes (v0.6)

### Project Command Semantics — COMPLETED

- Project operations are now grouped under `ctx project ...`
- Added `ctx project list`, `ctx project init <project>`, `ctx project onboard <project>`, and `ctx project delete <project>`
- `ctx project onboard <project> [--force] [--yes]` saves streamed markdown project context for an existing codebase without forcing the agent/user to hand-edit `project.md`
- `ctx init` remains the session bootstrap command; project creation is explicit through `ctx project init`
- README, CLI help, and bootstrap examples now prefer explicit `--project=<name>` usage for task operations

### Session Resume & Sync Preferences — COMPLETED

- Added `ctx resume` as the first command for a new working session
- Resume validates/repairs local storage, attempts sync when possible, and orients the user before task selection
- `ctx sync` now supports `--prefer=local|remote` and `--force` for explicit dirty-storage resolution
- Dirty local storage is reported as a local state problem, not a remote/auth problem
- Sync output now makes the preferred direction and outcome clearer for operational use

### Compact Context Output — COMPLETED

- Added `ctx context --compact` as an opt-in token-efficient context digest
- Compact output uses short stable keys such as `PRJ`, `FOCUS`, `TASKS`, `OBJ`, `STATE`, `NEXT`, `ISSUES`, `DECIDED`, `LAST`, `OPEN`, `RESP`, `DECISION`, and `SRC`
- Empty optional fields are omitted rather than rendered as placeholders
- Project-level compact context includes a concise `TASKS:` line when no task focus is selected
- `templates/bootstrap.md` now instructs agents to start with compact context and fall back to full `ctx context` only when needed
- Bootstrap wording explicitly tells tool-based agents to run `ctx ...` as shell/terminal commands, not runtime tool calls such as `ctx:context`
- Real-agent validation: OpenCode now follows the shell-command path after the wording clarification

### History, Timeline, Search, And Diff — COMPLETED

- Added `ctx history [--project=<name>] [--task=<name>] [--limit=<n>] [--since=<duration>]`
  - renders a narrative story from snapshots, handoffs, and activity events
- Added `ctx timeline [--project=<name>] [--task=<name>] [--limit=<n>] [--since=<duration>]`
  - prints a raw chronological activity timeline
- Added `ctx search [--project=<name>] [--task=<name>] [--limit=<n>] [--since=<duration>] <query>`
  - searches snapshots, handoffs, and collaboration artifacts
  - supports newest-N limiting and relative time windows like `24h` or `7d`
- Added `ctx diff <task> [<from-snapshot> <to-snapshot>] [--project=<name>]`
  - compares the latest two snapshots by default
  - supports explicit snapshot filename pairs

### Typed Collaboration Artifacts — COMPLETED

- Added `ctx capture --type=state|proposal|request|response|decision`
- Default `state` preserves snapshot behavior; non-state captures write separate typed artifacts
- `ctx context` surfaces compact collaboration summaries without treating proposals/requests as authoritative state
- Added `ctx artifact list <task>`, `ctx artifact show <task> <filename>`, and `ctx resolve <task> <filename>`
- Resolved collaboration artifacts move under the task `resolved/` directory and are removed from open counts
- Decision captures can link to and resolve proposal/request artifacts with `--resolves=<filename>`, appending `## Resolves` and moving the linked artifact to `resolved/`

### Activity Monitoring & Maintenance — COMPLETED

- Activity stream coverage now includes task writes, typed artifacts, task/project lifecycle operations, export/import, repair, agent install/remove, and sync
- `ctx watch --tui` supports interactive project filtering (`p` to cycle, `P` for all projects)
- `ctx repair --activity` provides explicit maintenance for `events/activity.ndjson`
- Exact duplicate lines are removed, malformed remnants are dropped, and valid events are reordered by timestamp

### Release & Distribution — COMPLETED

- Added GoReleaser configuration for multi-platform release artifacts
- Added GitHub Actions release workflow for `v*` tags
- Added `install.sh` direct-install fallback for macOS/Linux
- `ctx --version` / `ctx version` expose injected build metadata (`version`, `commit`, `date`) in release builds

---

## Recent Changes (v0.5)

### Typed Capture Artifacts — COMPLETED

- Added `ctx capture --type=state|proposal|request|response|decision`; default `state` preserves existing snapshot behavior
- Non-state captures write separate `proposal.*.md`, `request.*.md`, `response.*.md`, or `decision.*.md` files and preserve the raw markdown body with metadata
- `ctx context` continues to derive objective/current state/next step from snapshots only, then adds compact collaboration summaries for open proposals, requests, latest response, and latest decision
- Updated `templates/bootstrap.md` so agents use `--type=state` only for authoritative task state and typed captures for proposals, requests, responses, and decisions
- Reinstalled bootstrap instructions can now guide concurrent agents to separate coordination artifacts from operational state

### Visibility Watch Mode — COMPLETED

- Added `ctx watch [--project=<name>] [--interval=<duration>]`
- Watches a shared append-only activity stream and prints readable lifecycle events in a live feed
- Designed as the lightweight visibility MVP before any future TUI
- `ctx watch --tui` now supports interactive project filtering (`p` to cycle, `P` for all projects)
- Activity stream coverage now includes task writes, task/project lifecycle operations, export/import, repair, agent install/remove, and sync
- Added tests for watch summaries and updated README/help text
- `ctx repair --activity` now provides explicit maintenance for the shared activity log by deduplicating exact duplicate lines, dropping malformed remnants, and reordering valid entries by timestamp
- Future consideration: keep the activity log as a single file for now; revisit log splitting only if file size or history/watch/repair performance becomes a real bottleneck

### Append-Only Storage Model — COMPLETED

- New `internal/filestore` package: `NewSnapshotName()`, `NewHandoffName()`, `AtomicWrite()`, `LatestSnapshot()`, `LatestHandoff()`, `AllSnapshots()`, `AllHandoffs()`
- Every `ctx capture`, `ctx handoff`, `ctx snapshot refresh` now writes a new timestamped file (`snapshot.20060102T150405Z.a3f2c1.md`) instead of overwriting a fixed filename
- Writes are atomic: temp file + `rename(2)` — readers never see a partial write
- `ctx task start` no longer pre-creates `snapshot.md` / `handoff.md` — files are created on first write
- `ctx context` reads the latest snapshot (lexicographic = chronological) and adds `source snapshot: <name>` to output
- `ctx export` writes importable archives for task, project, or full-session scope depending on flags
- `ctx import` restores these exported archives through a single import path
- Breaking change: old fixed-name `snapshot.md` / `handoff.md` files are no longer read

### Task Lifecycle Status — COMPLETED

- Added per-task metadata with explicit `active` / `closed` status
- Added `ctx task close <task>` and `ctx task reopen <task>`
- `ctx list` now shows active tasks by default
- `ctx list --all` and `ctx list --status=<active|closed>` expose lifecycle filtering
- Project-level `ctx context` only considers active tasks when inferring current focus

### Archive Workflow Simplification — COMPLETED

- Removed the separate `share` command
- `ctx export` and `ctx import` are now symmetric for task archives
- Added project export with `ctx export --project=<name[,name2...]>`
- Added full-session export with `ctx export --session`
- Exported archives now carry a manifest so import can restore task, project, or session scope correctly

### VCS Package — COMPLETED

- New `internal/vcs` package: `VCS` interface + `Git` implementation via `exec.Command("git", ...)` — no external libraries
- `GitError` wraps failures without exposing raw git output to the user
- Git errors logged to `local/git.log` (best-effort, never blocks the caller)
- Operations: `Init`, `Clone`, `AddRemote`, `Commit` (filters non-existent files), `Push`, `Pull`, `Fsck`, `AbortInProgress`
- Full test suite; skipped automatically when git is not available

### Distribution & Release — COMPLETED

- Added `.github/workflows/release.yml` to trigger releases on `v*` tags
- Added `.goreleaser.yml` for multi-platform builds, archives, checksums, GitHub draft releases, and Homebrew tap publishing
- Added `install.sh` as a direct install fallback for macOS/Linux users outside Homebrew
- Extended build metadata injection targets in `cmd/version.go` so release builds can carry version/commit/date data
- Updated README with distribution targets, release flow, install fallback, and release checklist

---

## Recent Changes (v0.4)

### Capture Bridge Completion - COMPLETED

- Added `ctx capture <task> --yes` — skips interactive confirmation, allowing save in non-interactive environments where `stdin` is piped and `/dev/tty` is unavailable
- `ctx capture` now cleanly separates content source from confirmation mode:
  - piped stdin is used as the proposed snapshot update
  - `--yes` auto-confirms after showing the proposed summary
  - interactive mode still asks `y/n` before writing
- Added tests for `capture` argument parsing and for piped stdin with `--yes`

### Agent Bootstrap Removal - COMPLETED

- Exposed `ctx agent remove [--project=<name>]` in `cmd/main.go`
- Updated CLI help and examples

### Confinement Docs Alignment - COMPLETED

- Updated `README.md` with an explicit Agent Confinement Principle section
- Clarified that agents must treat `ctx` as the only bridge to `~/.continuum/`
- Updated `templates/bootstrap.md` with both interactive and non-interactive `ctx capture` pipe examples
- Updated `templates/agent.md` to explicitly forbid direct access to `~/.continuum/`

### Init Template Source Override - COMPLETED

- Added `ctx init [<project>] --templates=<path>` to seed the first template set from an explicit directory
- Explicit `--templates` paths are now validated up front and fail fast if required files are missing
- Added tests for init arg parsing and explicit template source lookup/validation

### Init Force Semantics - COMPLETED

- Added `ctx init --project=<name>` as an alternative to positional project syntax
- Added `ctx init ... --force` to overwrite existing init seed files intentionally
- Default init behavior is now idempotent for existing files; project seed files are preserved unless `--force` is passed

---

## Recent Changes (v0.3)

### Final Cleanup - COMPLETED

- Fixed `SnapshotRefresh`: all `promptWithDefault` errors were silently ignored with `_`; now propagated
- Removed `--prompt-only` flag from `ctx context` — was a no-op (legacy from `resume`); removed from handler and help text
- Updated README: corrected `ctx context <task>` syntax, updated storage tree with `agent-targets.txt`, added customization note

### Test Coverage Expansion - COMPLETED

- `internal/parse` — `Sections()`, `CleanValue()` including edge cases
- `internal/setup` — `ContinuumPath()`, `ListProjects()`, `DetectProject()`
- `internal/task` — `Start()` (creates files, idempotent, empty name), `List()`
- `internal/agent` — `loadTargetFiles()` (missing, empty, valid), `installToFile()` (skip, not found), `removeFromFile()`
- `internal/template` — `findTemplate()` (found, not found), `GetBootstrap()` (placeholder substitution), `InitTemplates()`

All 7 packages now have test files. Zero test failures across all packages.

### Deduplication & Dead Code Removal - COMPLETED

- Removed `internal/render` package — became unused after the legacy raw-context resume flow was removed
- Extracted `internal/parse` package with `Sections()` and `CleanValue()` — eliminates identical duplicate functions that existed independently in `internal/context` (parser.go, package.go) and `internal/task` (utils.go)
- `context.ParseSections` and `context.cleanValue` now delegate to `parse`
- `task.parseSections` and `task.cleanPrefill` now delegate to `parse`

### Bootstrap & Command Cleanup - COMPLETED

- Removed the legacy raw-context behavior from `ctx resume`; project/task context is now delivered by `ctx context` (distilled output, ADR 007)
- Current `ctx resume` remains a storage/session command: validate storage health, repair safe intermediate git states, sync when possible, and orient before project selection
- Removed unused `render` package import from `cmd/main.go`
- Updated `templates/bootstrap.md`:
  - Added `ctx list` to step 1 — agent can discover available tasks before calling `ctx capture`
  - Added step 4 for `ctx task start` — agent may create tasks when user intent is clear
  - Added placeholder comment warning (`do not remove %s placeholders`)
  - Renumbered steps for consistency

### Configurable Agent Targets - COMPLETED

- Removed hardcoded `targetFiles` list from `internal/agent/install.go`
- Added `templates/agent-targets.txt` — one filename per line, `#` comments supported
- `ctx init` creates `~/.continuum/agent-targets.txt` from template (skips if already exists)
- `ctx agent install` and `ctx agent remove` now read targets exclusively from the config file
- Clear error if file is missing: "run 'ctx init' first"
- Users can add any agent instruction file (`.cursorrules`, `.windsurfrules`, `GEMINI.md`, etc.)

### Code Quality & Safety Fixes - COMPLETED

#### cmd/main.go
- Added `parseFlag(arg, prefix)` helper — prevents panic on flags like `--project=` with no value (was unsafe `arg[N:]` slicing)
- Added `die(err)` helper — all errors now go to `os.Stderr`; previously all went to stdout, breaking pipe/script usage
- Eliminated 18 duplicated `if err != nil { fmt.Println; os.Exit }` patterns
- Eliminated 9 duplicated `--project=` flag parsing blocks

#### internal/export/export.go
- Fixed incomplete `os.Stat` error checks in export/archive paths — previously only checked `IsNotExist`, silently ignoring permission errors and other OS errors
- Fixed all `os.MkdirAll` calls that silently ignored errors — now propagated via `resolveOutputPath` and extraction helpers
- Fixed zip slip vulnerability in import — zip entries with paths like `../../etc/passwd` are now rejected
- Fixed absolute-path extraction vulnerability in import
- Import now restores generic export archives through a single path instead of task-only assumptions

#### Test coverage (from zero)
- `internal/context/context_test.go` — `extractField`, `extractConstraints`, `BuildContextPackage` (with/without task, constraints cap)
- `internal/export/export_test.go` — zip slip prevention, absolute path traversal, `extractTaskName` (from snapshot and filename), `resolveOutputPath`

#### internal/task/capture.go
- `ReadString('\n')` error now properly propagated; EOF treated as empty input (Ctrl+D / pipe)

#### internal/template/loader.go
- `userTemplatesDir`: `os.UserHomeDir()` error now handled with fallback chain (`HOME` → `USERPROFILE` → `"."`)
- `repoTemplatesDir`: `os.Executable()` error now handled with fallback to relative `"templates"`

#### internal/context/context.go — extractStackCore
- Removed hardcoded technology list (Laravel, Filament, Livewire, Tailwind, Redis, Pest) — now extracts any entry from the Stack section of project.md

#### internal/setup/init.go
- Extracted `initBase()` — eliminated duplication between `Init` and `InitSession` (~40 lines removed)
- Template loading errors now printed to stderr as warnings instead of being silently ignored

#### internal/context/context.go
- Fixed constraints rendering bug: when `len > 6`, the original loop added nothing (inner condition always false) and the "rebuild" fragily removed all lines with "- " prefix; replaced with a direct implementation
- Removed custom helpers `splitLines`, `trimSpace`, `stripPrefix` — replaced with `strings.Split`, `strings.TrimSpace`, `strings.TrimPrefix`

---

## Recent Changes (v0.2)

### ADR 007: Distilled Context Output - COMPLETED

- Rewrote BuildContextPackage to produce distilled output per ADR 007
- Output now includes: PROJECT, WORKING STYLE, CURRENT FOCUS, OBJECTIVE, CURRENT STATE, LOCKED DECISIONS, LAST SESSION, NEXT STEP, OPEN ISSUES
- Removed raw markdown headings and verbatim file dumps

### Conversational Model Implementation - COMPLETED

- Add CONTINUUM_PATH env var with ~/.continuum fallback
- Add ctx capture command for state summarization (y/e/n confirmation)
- Add ctx context command for agent context dump
- Add ctx agent install for bootstrap injection into AGENTS.md/CLAUDE.md
- Add templates in repo (profile, project, bootstrap, agent)
- Refactor init: ctx init (session) and ctx init <project> (session+project)
- Templates are modifiable by user in ~/.continuum/templates/
- Idempotent bootstrap injection with <!-- CONTINUUM:START --> markers

**Committed**: `94dfdd0` - feat: implement conversational model with templates

### Bug Fixes - COMPLETED

- Use absolute path for continuum storage (not relative .continuum/)
- Add --help support
- Fix templates command

**Committed**: `6b640fc`, `e9bedb5`, `a69aa8f`

---

## Version History

### v0.1 - Initial Release

- ctx init
- ctx task start <task>
- ctx list
- legacy ctx resume <task>
- legacy ctx resume <task> --prompt-only
- ctx export <task>
- ctx export --project=<name>
- ctx export --session
- ctx task close <task>
- ctx task reopen <task>
- ctx handoff <task>
- ctx snapshot refresh <task>
- ctx import <zip>
- ctx use <project>
- Encryption support (AES-256-GCM)

---

## Architecture

### Storage Location

```
~/.continuum/                    # git repo root (branch: main) — Phase 3
├── .git/
├── .gitignore
├── profile.md
├── agent-targets.txt
├── skills/
│   └── agent.md
├── projects/
│   └── <project>/
│       ├── project.md
│       └── tasks/
│           └── <task>/
│               ├── snapshot.20060102T150405Z.a3f2c1.md   # append-only
│               ├── handoff.20060102T150405Z.b7c1e4.md    # append-only
│               ├── proposal.20060102T150405Z.c91a8f.md   # typed collaboration artifact
│               ├── request.20060102T150405Z.d71b43.md    # typed collaboration artifact
│               ├── response.20060102T150405Z.e9b802.md   # typed collaboration artifact
│               ├── decision.20060102T150405Z.f1432a.md   # typed collaboration artifact
│               ├── resolved/                             # resolved collaboration artifacts
│               └── notes.md
├── events/
│   └── activity.ndjson           # shared append-only activity stream
├── templates/
├── local/                       # not tracked by git
│   ├── unsynced                 # commit hashes pending push
│   └── git.log                  # git error log (rotating)
└── exports/                     # not tracked by git
```

### Templates

Templates are in the repo under `templates/`:
- profile.md
- project.md
- bootstrap.md
- agent.md

Copied to ~/.continuum/templates/ on init. Edit these to customize defaults.

### Agent Bootstrap

Injected into AGENTS.md, CLAUDE.md, agent.md with markers:
```
<!-- CONTINUUM:START -->
...
<!-- CONTINUUM:END -->
```

Idempotent - safe to run multiple times.

---

## Design Principles

1. **Local-first**: All data stays in ~/.continuum/
2. **Privacy-aware**: Not designed for Git commit
3. **Agent-agnostic**: Works with any coding agent
4. **Offline-capable**: No cloud dependency
5. **Conversational**: User speaks naturally, agent summarizes, system distills
6. **User control**: Explicit confirmation before any persistence
