package task

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"continuum/internal/setup"
)

func TestSnapshotRefresh_AutoConfirmWithPipedStdin(t *testing.T) {
	base := withTempContinuum(t)
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Objective
Refresh snapshot

## Current State
- Refreshed from agent

## Next Step
- Run full validation

## Active Issues
- None
`)

	if err := SnapshotRefresh("my-task", "my-project", true); err != nil {
		t.Fatalf("SnapshotRefresh() error: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "my-project", "tasks", "my-task")
	content := readLatestFileInDir(t, taskDir, "snapshot.")
	for _, want := range []string{
		"Refresh snapshot",
		"- Refreshed from agent",
		"- Run full validation",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected snapshot to contain %q\ncontent:\n%s", want, content)
		}
	}
}

func TestSnapshotRefresh_CommitsFileWhenGitRepoExists(t *testing.T) {
	base := withTempContinuum(t)
	if err := setup.Init("my-project", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Objective
Refresh snapshot

## Current State
- Git-backed refresh

## Next Step
- Verify log
`)

	if err := SnapshotRefresh("my-task", "my-project", true); err != nil {
		t.Fatalf("SnapshotRefresh() error: %v", err)
	}

	out, err := exec.Command("git", "-C", base, "log", "--format=%s", "-1").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "capture(my-project/my-task): snapshot refreshed" {
		t.Fatalf("unexpected git commit message: %q", strings.TrimSpace(string(out)))
	}
}
