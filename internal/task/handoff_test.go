package task

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"continuum/internal/setup"
)

func TestHandoff_AutoConfirmWithPipedStdin(t *testing.T) {
	base := withTempContinuum(t)
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Objective
Prepare handoff

## What Was Done
- Added bridge support

## Current State
- Ready for next agent

## Next Recommended Step
- Validate in real repo

## Unresolved Questions
- None
`)

	if err := Handoff("my-task", "my-project", true); err != nil {
		t.Fatalf("Handoff() error: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "my-project", "tasks", "my-task")
	content := readLatestFileInDir(t, taskDir, "handoff.")
	for _, want := range []string{
		"Prepare handoff",
		"- Added bridge support",
		"- Ready for next agent",
		"- Validate in real repo",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected handoff to contain %q\ncontent:\n%s", want, content)
		}
	}
}

func TestHandoff_CommitsFileWhenGitRepoExists(t *testing.T) {
	base := withTempContinuum(t)
	if err := setup.Init("my-project", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Objective
Prepare handoff

## What Was Done
- Added git-backed handoff

## Current State
- Ready
`)

	if err := Handoff("my-task", "my-project", true); err != nil {
		t.Fatalf("Handoff() error: %v", err)
	}

	out, err := exec.Command("git", "-C", base, "log", "--format=%s", "-1").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "handoff(my-project/my-task): handoff created" {
		t.Fatalf("unexpected git commit message: %q", strings.TrimSpace(string(out)))
	}
}
