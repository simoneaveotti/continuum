# Storage Patterns

Continuum stores all state in a local directory (default: `~/.ctx/`).
`CONTINUUM_PATH` overrides this location.

Because each `CONTINUUM_PATH` is an independent storage instance with its own git repository, remote, and project set, you can run multiple isolated Continuum instances side by side with different sync policies and access rules.

## Pattern 1 — Single storage, no remote

Everything is local. No sync. No cloud dependency.

```bash
ctx init
ctx project init my-project
ctx context --project=my-project
```

Use when: single machine, personal projects, getting started.

## Pattern 2 — Single storage with remote

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

## Pattern 3 — Multiple storage instances for sensitive projects

Run a second Continuum instance in a different directory:

```bash
# Normal projects — synced
ctx context --project=my-app

# Sensitive project — local only
CONTINUUM_PATH=~/.ctx-private ctx context --project=sensitive-project
```

The sensitive storage has no remote configured. Agents working there must be bootstrapped with the correct `CONTINUUM_PATH`.

Use when: regulated, healthcare, client-confidential, or otherwise non-syncable work.

## Pattern 4 — Multiple storage instances with separate remotes

Each storage instance can point to a different remote:

```text
~/.ctx/          -> github.com/you/ctx-work
~/.ctx-private/  -> gitea.yourserver.com/you/ctx-private
~/.ctx-client/   -> github.com/you/ctx-client-x
```

Each instance stays independent: different projects, different remotes, different sync rules.

Use when: multi-client consulting, residency separation, or strict project boundaries.

## Pattern 5 — Work / personal separation

Common setup for employed developers or consultants:

```text
~/.ctx/          -> company remote
~/.ctx-personal/ -> personal remote
```

You can switch automatically with project-specific shell config such as:

```bash
# in .envrc for personal projects
export CONTINUUM_PATH=~/.ctx-personal
```

Use when: company and personal work must stay fully separate.

## Security Model

Continuum does not enforce access control between agents. Separation here is path isolation: an agent that does not know a `CONTINUUM_PATH` value cannot access that storage instance.

This is path-level isolation, not a cryptographic guarantee. For stronger local guarantees, keep sensitive storage unsynced and lock down directory permissions, for example:

```bash
chmod 700 ~/.ctx-private
```

Agents working on sensitive storage should be bootstrapped with the correct `CONTINUUM_PATH` explicitly set in their environment or launcher.
