package vcs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Git implements VCS using the system git binary.
type Git struct {
	logPath    string // path to git.log; empty = no logging
	authorName string // user name for git commits (empty = resolve from git config)
	authorEmail string // user email for git commits (empty = resolve from git config)
}

const gitLogMaxBytes = 1 << 20

// NewGit returns a Git instance. logPath is the file to append operation
// errors to; pass "" to disable logging.
func NewGit(logPath string) *Git {
	return &Git{logPath: logPath}
}

// SetIdentity overrides the author/committer identity used in git commits.
// When not set, the identity is resolved lazily from the repo's git config.
func (g *Git) SetIdentity(name, email string) {
	g.authorName = name
	g.authorEmail = email
}

// ensureIdentity resolves the git identity from the repository config if not
// already set. Falls back to "Continuum <continuum@local>".
func (g *Git) ensureIdentity(path string) {
	if g.authorName != "" {
		return
	}
	g.authorName = "Continuum"
	g.authorEmail = "continuum@local"
	if out, err := exec.Command("git", "-C", path, "config", "user.name").Output(); err == nil {
		if n := strings.TrimSpace(string(out)); n != "" {
			g.authorName = n
		}
	}
	if out, err := exec.Command("git", "-C", path, "config", "user.email").Output(); err == nil {
		if e := strings.TrimSpace(string(out)); e != "" {
			g.authorEmail = e
		}
	}
}

func (g *Git) Init(path string) error {
	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		return nil // already initialized
	}
	return g.run(path, "init", "-b", "main")
}

func (g *Git) Clone(url, path string) error {
	return g.runAt("", "clone", url, path)
}

func (g *Git) AddRemote(path, name, url string) error {
	// Skip if remote already exists
	out, _ := g.remoteURLNoLog(path, name)
	if strings.TrimSpace(out) != "" {
		return nil
	}
	return g.run(path, "remote", "add", name, url)
}

func (g *Git) HasRemote(path, name string) bool {
	out, err := g.remoteURLNoLog(path, name)
	return err == nil && strings.TrimSpace(out) != ""
}

func (g *Git) Commit(path, message string, files []string) error {
	if files != nil {
		// Only stage files that actually exist — absent files are staged via `git add -u`.
		var toAdd []string
		var toUpdate []string
		for _, f := range files {
			_, statErr := os.Stat(filepath.Join(path, f))
			if statErr == nil {
				toAdd = append(toAdd, f)
				continue
			}
			if os.IsNotExist(statErr) {
				toUpdate = append(toUpdate, f)
				continue
			}
			return fmt.Errorf("stat %s: %w", f, statErr)
		}
		if len(toAdd) > 0 {
			addArgs := append([]string{"add", "--"}, toAdd...)
			if err := g.run(path, addArgs...); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
		}
		if len(toUpdate) > 0 {
			updateArgs := append([]string{"add", "-u", "--"}, toUpdate...)
			if err := g.run(path, updateArgs...); err != nil {
				if !isPathspecError(err) {
					return fmt.Errorf("git add -u: %w", err)
				}
			}
		}
	}
	// Check if anything is actually staged before committing
	statusOut, _ := g.output(path, "diff", "--cached", "--name-only")
	if strings.TrimSpace(statusOut) == "" {
		return nil
	}
	if err := g.run(path, "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

func (g *Git) Push(path string) error {
	return g.run(path, "push", "origin", "main")
}

func (g *Git) Pull(path string) error {
	return g.run(path, "pull", "--rebase", "origin", "main", "--quiet")
}

func (g *Git) Fsck(path string) (bool, error) {
	err := g.run(path, "fsck")
	if err != nil {
		return false, nil // fsck found problems — not a command error
	}
	return true, nil
}

func (g *Git) AbortInProgress(path string) error {
	// Try rebase abort first, then merge abort — ignore errors from both
	// since only one (at most) will be in progress.
	rebasePath := filepath.Join(path, ".git", "rebase-merge")
	mergePath := filepath.Join(path, ".git", "MERGE_HEAD")

	if _, err := os.Stat(rebasePath); err == nil {
		if err := g.run(path, "rebase", "--abort"); err != nil {
			return fmt.Errorf("git rebase --abort: %w", err)
		}
		return nil
	}
	if _, err := os.Stat(mergePath); err == nil {
		if err := g.run(path, "merge", "--abort"); err != nil {
			return fmt.Errorf("git merge --abort: %w", err)
		}
	}
	return nil
}

func (g *Git) HeadCommit(path string) (string, error) {
	out, err := g.runAndCapture(path, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (g *Git) RevListCount(path string, args ...string) (int, error) {
	out, err := g.runAndCapture(path, append([]string{"rev-list", "--count"}, args...)...)
	if err != nil {
		return 0, err
	}
	count, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, fmt.Errorf("git rev-list count: %w", err)
	}
	return count, nil
}

func (g *Git) RemoteURL(path, name string) (string, error) {
	out, err := g.runAndCapture(path, "remote", "get-url", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (g *Git) Execute(path string, args ...string) error {
	return g.run(path, args...)
}

// run executes git <args> with working directory path.
func (g *Git) run(path string, args ...string) error {
	_, err := g.runAndCapture(path, args...)
	return err
}

// runAt executes git <args> with an arbitrary working directory (can be "").
func (g *Git) runAt(dir string, args ...string) error {
	if dir != "" {
		g.ensureIdentity(dir)
	}
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = g.gitEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.logError(strings.Join(args, " "), err, string(out))
		return &GitError{Op: strings.Join(args[:1], ""), Stderr: string(out)}
	}
	return nil
}

// output runs git <args> and returns stdout as a string. Errors are ignored.
func (g *Git) output(path string, args ...string) (string, error) {
	out, err := g.runAndCapture(path, args...)
	return out, err
}

func (g *Git) remoteURLNoLog(path, name string) (string, error) {
	g.ensureIdentity(path)
	cmd := exec.Command("git", "remote", "get-url", name)
	cmd.Dir = path
	cmd.Env = g.gitEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (g *Git) runAndCapture(path string, args ...string) (string, error) {
	g.ensureIdentity(path)
	cmd := exec.Command("git", args...)
	cmd.Dir = path
	cmd.Env = g.gitEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.logError(strings.Join(args, " "), err, string(out))
		return "", &GitError{Op: args[0], Stderr: string(out)}
	}
	return string(out), nil
}

func (g *Git) logError(op string, err error, stderr string) {
	if g.logPath == "" {
		return
	}
	g.rotateLog()
	// Best-effort append to log file — never block on log failure
	f, ferr := os.OpenFile(g.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if ferr != nil {
		return
	}
	defer f.Close()
	exitCode := "?"
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = fmt.Sprintf("%d", exitErr.ExitCode())
	}
	fmt.Fprintf(f, "%s git %s exit=%s stderr=%s\n",
		logTimestamp(), op, exitCode, strings.ReplaceAll(stderr, "\n", " "))
}

func (g *Git) rotateLog() {
	if g.logPath == "" {
		return
	}
	info, err := os.Stat(g.logPath)
	if err != nil {
		return
	}
	if info.Size() < gitLogMaxBytes {
		return
	}
	_ = os.Rename(g.logPath, g.logPath+".1")
}

// GitError is returned when a git command exits with a non-zero status.
// It never exposes the raw git output to the user — callers should wrap
// with a human-readable message.
type GitError struct {
	Op     string
	Stderr string
}

func (e *GitError) Error() string {
	return fmt.Sprintf("git %s failed", e.Op)
}

// IsGitError returns true if err is a *GitError.
func IsGitError(err error) bool {
	_, ok := err.(*GitError)
	return ok
}

func (g *Git) gitEnv() []string {
	env := os.Environ()
	env = append(env,
		"GIT_AUTHOR_NAME="+g.authorName,
		"GIT_AUTHOR_EMAIL="+g.authorEmail,
		"GIT_COMMITTER_NAME="+g.authorName,
		"GIT_COMMITTER_EMAIL="+g.authorEmail,
	)
	return env
}

func isPathspecError(err error) bool {
	if ge, ok := err.(*GitError); ok {
		return strings.Contains(ge.Stderr, "did not match any files")
	}
	return false
}
