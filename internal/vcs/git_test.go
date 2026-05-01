package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-b", "main", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()
	return dir
}

func TestGit_Init_Fresh(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	g := NewGit("")
	if err := g.Init(dir); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Error("expected .git directory to exist after Init")
	}
}

func TestGit_Init_Idempotent(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	g := NewGit("")
	if err := g.Init(dir); err != nil {
		t.Fatalf("first Init() error: %v", err)
	}
	if err := g.Init(dir); err != nil {
		t.Fatalf("second Init() should be no-op, got error: %v", err)
	}
}

func TestGit_AddRemote_SkipIfExists(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := initRepo(t)
	g := NewGit("")

	if err := g.AddRemote(dir, "origin", "https://example.com/repo.git"); err != nil {
		t.Fatalf("AddRemote() error: %v", err)
	}
	// Second call must not fail
	if err := g.AddRemote(dir, "origin", "https://example.com/repo.git"); err != nil {
		t.Fatalf("AddRemote() second call error: %v", err)
	}
}

func TestGit_Commit(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := initRepo(t)
	g := NewGit("")

	// Write a file and commit it
	filePath := filepath.Join(dir, "test.md")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := g.Commit(dir, "test: initial", []string{"test.md"}); err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	// Verify commit exists
	out, err := exec.Command("git", "-C", dir, "log", "--oneline").Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected at least one commit in log")
	}
}

func TestGit_Commit_NothingToCommit(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := initRepo(t)
	g := NewGit("")

	// File doesn't exist — nothing to stage, should not error
	if err := g.Commit(dir, "test: empty", []string{"nonexistent.md"}); err != nil {
		t.Fatalf("Commit() with nothing staged should not error, got: %v", err)
	}
}

func TestGit_Commit_EmptyFiles(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := initRepo(t)
	g := NewGit("")

	if err := g.Commit(dir, "test: no files", []string{}); err != nil {
		t.Fatalf("Commit() with empty file list should not error, got: %v", err)
	}
}

func TestGit_Fsck_DoesNotError(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := initRepo(t)
	g := NewGit("")

	// Fsck must never return an error — only (clean bool, nil)
	_, err := g.Fsck(dir)
	if err != nil {
		t.Fatalf("Fsck() returned unexpected error: %v", err)
	}
}

func TestGit_AbortInProgress_NoOp(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := initRepo(t)
	g := NewGit("")

	// No rebase or merge in progress — should not error
	if err := g.AbortInProgress(dir); err != nil {
		t.Fatalf("AbortInProgress() on clean repo error: %v", err)
	}
}

func TestGitError_NeverExposesRawOutput(t *testing.T) {
	e := &GitError{Op: "push", Stderr: "fatal: repository not found\n[secret internal detail]"}
	msg := e.Error()
	if msg != "git push failed" {
		t.Errorf("GitError.Error() = %q, want %q", msg, "git push failed")
	}
}

func TestIsGitError(t *testing.T) {
	e := &GitError{Op: "pull"}
	if !IsGitError(e) {
		t.Error("IsGitError() = false for *GitError")
	}
	if IsGitError(nil) {
		t.Error("IsGitError(nil) = true")
	}
}
