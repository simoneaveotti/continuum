// Package vcs provides a minimal git wrapper used internally by Continuum.
// All git operations are performed via exec.Command("git", ...) — no external
// libraries. The interface is intentionally narrow: only operations Continuum
// actually needs are exposed.
//
// Failure contract:
//   - Blocking errors (Init, Clone, Commit) return non-nil error.
//   - Degraded operations (Push, Pull) return error but the caller is expected
//     to log, update the unsynced marker, and continue — never block the user.
//   - Git stack traces are never surfaced to the user. All errors are wrapped
//     with human-readable context.
package vcs

// VCS is the interface for version-control operations used by Continuum.
type VCS interface {
	// Init initializes a new git repository at path with branch "main".
	// No-op if .git/ already exists.
	Init(path string) error

	// Clone clones url into path. Fails if path is non-empty.
	Clone(url, path string) error

	// AddRemote adds a named remote to the repo at path.
	// No-op if the remote already exists.
	AddRemote(path, name, url string) error

	// HasRemote reports whether the named remote exists in the repo at path.
	HasRemote(path, name string) bool

	// Commit stages files and creates a commit in the repo at path.
	// files must be relative to path.
	Commit(path, message string, files []string) error

	// Push pushes to origin main. Best-effort: errors are returned but
	// callers must treat them as degraded, not blocking.
	Push(path string) error

	// Pull runs git pull --rebase origin main. Non-blocking on failure:
	// callers must treat failure as degraded and read local state.
	Pull(path string) error

	// Fsck runs git fsck. Returns clean=true if no problems found.
	Fsck(path string) (clean bool, err error)

	// AbortInProgress aborts any in-progress rebase or merge.
	AbortInProgress(path string) error

	// HeadCommit returns the current HEAD commit hash.
	HeadCommit(path string) (string, error)

	// RevListCount runs git rev-list --count with the provided arguments.
	RevListCount(path string, args ...string) (int, error)

	// RemoteURL returns the URL configured for the named remote.
	RemoteURL(path, name string) (string, error)

	// Execute runs an arbitrary git command.
	Execute(path string, args ...string) error
}
