# Git Backend Rollout Checklist

This checklist validates real-world usage of Continuum's git-backed storage.

Run the steps in order.

## 1. Local Bootstrap and Empty Remote Attach

```bash
export CONTINUUM_PATH="$(mktemp -d)"
mkdir -p /tmp/continuum-remote.git
git init --bare /tmp/continuum-remote.git

./ctx init
./ctx sync --remote=/tmp/continuum-remote.git
```

Expected:

- `origin` is configured as the remote
- the empty remote is bootstrapped with branch `main`
- sync output reports an initial push and zero pulled commits

## 2. Project, Task, and Capture

```bash
./ctx init continuum
./ctx task start smoke
./ctx capture smoke --yes <<'EOF'
## Objective
Validate git backend rollout

## Current State
- created smoke task

## Next Step
- run sync

## Constraints
- local test only

## Active Issues
- none
EOF
```

Expected:

- a new snapshot file is written
- a local git commit is created
- if the remote is reachable, the best-effort push succeeds

## 3. Normal Sync Against a Populated Remote

```bash
./ctx sync
```

Expected:

- no bootstrap message is shown
- sync output reports reasonable push/pull counts, often `0 / 0`

## 4. Simulate a Remote Commit from a Second Machine

```bash
git clone /tmp/continuum-remote.git /tmp/continuum-peer
cd /tmp/continuum-peer
git config user.email test@example.com
git config user.name test
echo remote-change > note.txt
git add note.txt
git commit -m "test: remote change"
git push origin main
cd -

./ctx sync
```

Expected:

- sync reports `Pull: 1 commit` or equivalent
- no error is shown
- the local storage is updated with the remote commit

## 5. Offline / Unsynced Behavior

Temporarily make the remote unreachable, then run:

```bash
./ctx capture smoke --yes <<'EOF'
## Objective
Validate unsynced handling

## Current State
- push is expected to fail

## Next Step
- run sync again later

## Constraints
- remote unavailable

## Active Issues
- unsynced marker expected
EOF
```

Expected:

- capture still succeeds locally
- `local/unsynced` is created
- `local/git.log` records a real git error

## 6. Recovery After Offline Work

Restore the remote, then run:

```bash
./ctx sync
```

Expected:

- the pending local commit is pushed successfully
- `local/unsynced` is cleared

## 7. Context Loading While the Remote Is Unreachable

With a configured but unreachable remote:

```bash
./ctx context --project=continuum
```

Expected:

- a warning about failed pull is shown
- context is still printed from local state

## 8. Snapshot Cleanup

Create several snapshots for the same task, then run:

```bash
./ctx snapshot clean smoke --keep=2
```

Expected:

- only the newest 2 `snapshot.*.md` files remain
- handoff files are untouched
- a cleanup commit is created

## 9. Auditable Deletes

```bash
./ctx task delete smoke
./ctx project delete continuum
```

Expected:

- the directories are removed
- matching git commits are created
- a later sync succeeds cleanly

## 10. Repair Smoke Test

```bash
./ctx repair
```

Expected:

- on a healthy repository, the command reports no issues
