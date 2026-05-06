package task

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"continuum/internal/filestore"
	"continuum/internal/setup"
)

func useStdinFile(t *testing.T, content string) {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatalf("CreateTemp() error: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("WriteString() error: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("Seek() error: %v", err)
	}

	oldStdin := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = oldStdin
		f.Close()
	})
}

func TestCapture_AutoConfirmWithPipedStdin(t *testing.T) {
	base := withTempContinuum(t)
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Objective
Ship --yes support

## Current State
- Parser updated

## Decisions
- Keep --yes non-interactive because agents need autonomous saves

## Next Step
- Add tests

## Constraints
- Keep CLI behavior stable

## Active Issues
- None
`)

	if err := Capture("my-task", "my-project", filestore.StateCapture, true); err != nil {
		t.Fatalf("Capture() error: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "my-project", "tasks", "my-task")
	content := readLatestFileInDir(t, taskDir, "snapshot.")
	for _, want := range []string{
		"Ship --yes support",
		"- Parser updated",
		"## Decisions (Locked)\n- Keep --yes non-interactive because agents need autonomous saves",
		"- Add tests",
		"- Keep CLI behavior stable",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected snapshot to contain %q\ncontent:\n%s", want, content)
		}
	}
}

func TestCapture_AcceptsCanonicalLockedDecisions(t *testing.T) {
	base := withTempContinuum(t)
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Objective
Preserve canonical decisions

## Current State
- Capture accepts canonical heading

## Decisions (Locked)
- Keep the canonical snapshot heading stable because ctx context already reads it

## Next Step
- Verify compact context
`)

	if err := Capture("my-task", "my-project", filestore.StateCapture, true); err != nil {
		t.Fatalf("Capture() error: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "my-project", "tasks", "my-task")
	content := readLatestFileInDir(t, taskDir, "snapshot.")
	if !strings.Contains(content, "## Decisions (Locked)\n- Keep the canonical snapshot heading stable because ctx context already reads it") {
		t.Fatalf("expected canonical decisions to survive\ncontent:\n%s", content)
	}
}

func TestCapture_CommitsSnapshotWhenGitRepoExists(t *testing.T) {
	base := withTempContinuum(t)
	if err := setup.Init("my-project", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Objective
Ship git history

## Current State
- Snapshot saved

## Next Step
- Verify commit
`)

	if err := Capture("my-task", "my-project", filestore.StateCapture, true); err != nil {
		t.Fatalf("Capture() error: %v", err)
	}

	out, err := exec.Command("git", "-C", base, "log", "--format=%s", "-1").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "capture(my-project/my-task): snapshot updated" {
		t.Fatalf("unexpected git commit message: %q", strings.TrimSpace(string(out)))
	}
}

func TestCapture_ProposalPreservesRawMarkdown(t *testing.T) {
	base := withTempContinuum(t)
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Proposal
Use --type for collaboration artifacts.

## Rationale
- Avoid new commands.
`)

	if err := Capture("my-task", "my-project", filestore.ProposalCapture, true); err != nil {
		t.Fatalf("Capture() error: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "my-project", "tasks", "my-task")
	content := readLatestFileInDir(t, taskDir, "proposal.")
	for _, want := range []string{
		"# TASK PROPOSAL",
		"## Capture Type\nproposal",
		"## Proposal\nUse --type for collaboration artifacts.",
		"## Rationale\n- Avoid new commands.",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected proposal to contain %q\ncontent:\n%s", want, content)
		}
	}
}

func TestCapture_StateIsolatedFromProposals(t *testing.T) {
	base := withTempContinuum(t)
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Objective
Keep state clean

## Current State
- Phase 1 complete

## Next Step
- Implement isolation
`)
	if err := Capture("my-task", "my-project", filestore.StateCapture, true); err != nil {
		t.Fatalf("state Capture() error: %v", err)
	}

	useStdinFile(t, `## Proposal
Alternative: rewrite in Rust.

## Rationale
- Faster, supposedly.
`)
	if err := Capture("my-task", "my-project", filestore.ProposalCapture, true); err != nil {
		t.Fatalf("proposal Capture() error: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "my-project", "tasks", "my-task")
	snaps, err := filestore.AllCapturesOfType(taskDir, filestore.StateCapture)
	if err != nil {
		t.Fatalf("AllCapturesOfType(state): %v", err)
	}
	proposals, err := filestore.AllCapturesOfType(taskDir, filestore.ProposalCapture)
	if err != nil {
		t.Fatalf("AllCapturesOfType(proposal): %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("expected 1 state snapshot, got %d", len(snaps))
	}
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	data, err := LoadCaptureData("my-task", "my-project")
	if err != nil {
		t.Fatalf("LoadCaptureData() error: %v", err)
	}
	if data.Objective != "Keep state clean" {
		t.Fatalf("state contaminated: objective = %q", data.Objective)
	}
	if strings.Contains(data.State, "rewrite in Rust") {
		t.Fatalf("state contaminated by proposal: %+v", data)
	}
}

func TestCapture_DecisionResolvesProposal(t *testing.T) {
	base := withTempContinuum(t)
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Proposal
Adopt linked decision resolution.
`)
	if err := Capture("my-task", "my-project", filestore.ProposalCapture, true); err != nil {
		t.Fatalf("proposal Capture() error: %v", err)
	}
	taskDir := filepath.Join(base, "projects", "my-project", "tasks", "my-task")
	proposals, err := filestore.AllCapturesOfType(taskDir, filestore.ProposalCapture)
	if err != nil {
		t.Fatalf("AllCapturesOfType(proposal): %v", err)
	}
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}
	proposalName := filepath.Base(proposals[0])

	useStdinFile(t, `## Decision
Adopt linked decision resolution.

## Rationale
- It closes the proposal loop.
`)
	if err := CaptureWithOptions("my-task", "my-project", CaptureOptions{
		Type:        filestore.DecisionCapture,
		AutoConfirm: true,
		Resolves:    proposalName,
	}); err != nil {
		t.Fatalf("decision CaptureWithOptions() error: %v", err)
	}

	proposals, err = filestore.AllCapturesOfType(taskDir, filestore.ProposalCapture)
	if err != nil {
		t.Fatalf("AllCapturesOfType(proposal) after decision: %v", err)
	}
	if len(proposals) != 0 {
		t.Fatalf("expected proposal to be resolved, got %d open proposal(s)", len(proposals))
	}
	if _, err := os.Stat(filepath.Join(taskDir, "resolved", proposalName)); err != nil {
		t.Fatalf("expected resolved proposal file: %v", err)
	}
	decision := readLatestFileInDir(t, taskDir, "decision.")
	for _, want := range []string{
		"## Decision\nAdopt linked decision resolution.",
		"## Resolves\n- " + proposalName,
	} {
		if !strings.Contains(decision, want) {
			t.Fatalf("expected decision to contain %q\ncontent:\n%s", want, decision)
		}
	}
}

func TestCapture_ResolvesRequiresDecision(t *testing.T) {
	withTempContinuum(t)
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	useStdinFile(t, `## Proposal
Invalid link.
`)
	err := CaptureWithOptions("my-task", "my-project", CaptureOptions{
		Type:        filestore.ProposalCapture,
		AutoConfirm: true,
		Resolves:    "proposal.20260416T095334Z.fdd7a5.md",
	})
	if err == nil {
		t.Fatal("expected --resolves with proposal capture to fail")
	}
	if !strings.Contains(err.Error(), "--type=decision") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCapture_NonStateWithoutStdinFails(t *testing.T) {
	if err := Capture("my-task", "my-project", filestore.ProposalCapture, true); err == nil {
		t.Fatal("expected non-state capture without stdin to fail")
	}
}

func TestCapture_CommitsProposalWhenGitRepoExists(t *testing.T) {
	base := withTempContinuum(t)
	if err := setup.Init("my-project", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	useStdinFile(t, `## Proposal
Use typed captures.
`)

	if err := Capture("my-task", "my-project", filestore.ProposalCapture, true); err != nil {
		t.Fatalf("Capture() error: %v", err)
	}

	out, err := exec.Command("git", "-C", base, "log", "--format=%s", "-1").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "proposal(my-project/my-task): proposal captured" {
		t.Fatalf("unexpected git commit message: %q", strings.TrimSpace(string(out)))
	}
}
