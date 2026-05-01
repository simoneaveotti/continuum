package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractField(t *testing.T) {
	content := `# TASK SNAPSHOT

## Objective
Build the auth module

## Current State
- Step 1 done
- Step 2 pending

## Next Step
Write tests
`
	tests := []struct {
		field string
		want  string
	}{
		{"objective", "Build the auth module"},
		{"next step", "Write tests"},
		{"missing", ""},
	}

	for _, tt := range tests {
		got := extractField(content, tt.field)
		if got != tt.want {
			t.Errorf("extractField(%q) = %q, want %q", tt.field, got, tt.want)
		}
	}
}

func TestExtractConstraints(t *testing.T) {
	content := `# PROJECT

## Constraints
- No breaking changes
- Keep it offline-first
- Must support Go 1.21+
`
	got := extractConstraints(content)
	if len(got) != 3 {
		t.Fatalf("expected 3 constraints, got %d: %v", len(got), got)
	}
	if got[0] != "No breaking changes" {
		t.Errorf("got[0] = %q, want %q", got[0], "No breaking changes")
	}
}

func TestExtractConstraintsEmpty(t *testing.T) {
	content := `# PROJECT

## Summary
Just a project
`
	got := extractConstraints(content)
	if len(got) != 0 {
		t.Errorf("expected 0 constraints, got %d", len(got))
	}
}

func TestBuildContextPackageConstraintsCap(t *testing.T) {
	// Build a project.md with 8 constraints — output must cap at 6
	var sb strings.Builder
	sb.WriteString("# PROJECT\n\n## Summary\ntest project\n\n## Constraints\n")
	for i := 1; i <= 8; i++ {
		sb.WriteString("- constraint ")
		sb.WriteString(string(rune('0' + i)))
		sb.WriteString("\n")
	}

	ctx := &ContextData{
		Profile: "# Profile\n",
		Project: sb.String(),
	}

	output := BuildContextPackage(ctx, "", "testproject")

	count := strings.Count(output, "\n- ")
	if count > 6 {
		t.Errorf("expected at most 6 constraint lines, got %d\noutput:\n%s", count, output)
	}
}

func TestBuildContextPackageNoTask(t *testing.T) {
	ctx := &ContextData{
		Profile: "# Profile\n",
		Project: "# PROJECT\n\n## Summary\nmy project\n",
	}

	output := BuildContextPackage(ctx, "", "myproject")

	for _, expected := range []string{"PROJECT: myproject", "CURRENT FOCUS:", "OBJECTIVE:", "NEXT STEP:"} {
		if !strings.Contains(output, expected) {
			t.Errorf("output missing %q\noutput:\n%s", expected, output)
		}
	}
}

func TestBuildContextPackageWithTask(t *testing.T) {
	snapshot := `# TASK SNAPSHOT

## Objective
Fix the login bug

## Next Step
Deploy to staging
`
	ctx := &ContextData{
		Profile: "# Profile\n",
		Project: "# PROJECT\n\n## Summary\nmy project\n",
		TaskContexts: map[string]*ContextData{
			"login-fix": {Snapshot: snapshot},
		},
	}

	output := BuildContextPackage(ctx, "login-fix", "myproject")

	for _, expected := range []string{
		"CURRENT FOCUS: login-fix",
		"OBJECTIVE: Fix the login bug",
		"NEXT STEP: Deploy to staging",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("output missing %q\noutput:\n%s", expected, output)
		}
	}
}

func TestBuildContextPackageWithDirectTaskContext(t *testing.T) {
	snapshot := `# TASK SNAPSHOT

## Objective
Fix the login bug

## Current State
- Patch applied

## Next Step
Deploy to staging
`
	ctx := &ContextData{
		Profile:  "# Profile\n",
		Project:  "# PROJECT\n\n## Summary\nmy project\n",
		Snapshot: snapshot,
		Handoff:  "# TASK HANDOFF\n",
	}

	output := BuildContextPackage(ctx, "login-fix", "myproject")

	for _, expected := range []string{
		"CURRENT FOCUS: login-fix",
		"OBJECTIVE: Fix the login bug",
		"CURRENT STATE: Patch applied",
		"NEXT STEP: Deploy to staging",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("output missing %q\noutput:\n%s", expected, output)
		}
	}
}

func TestBuildContextPackageSingleTaskBecomesImplicitFocus(t *testing.T) {
	snapshot := `# TASK SNAPSHOT

## Objective
Validate bootstrap flow

## Current State
- Claude is reading context correctly

## Next Step
Run agnostic agent test
`
	ctx := &ContextData{
		Profile: "# Profile\n",
		Project: "# PROJECT\n\n## Summary\nmy project\n",
		TaskContexts: map[string]*ContextData{
			"continuum-bootstrap-test": {Snapshot: snapshot},
		},
	}

	output := BuildContextPackage(ctx, "", "myproject")

	for _, expected := range []string{
		"CURRENT FOCUS: continuum-bootstrap-test",
		"OBJECTIVE: Validate bootstrap flow",
		"CURRENT STATE: Claude is reading context correctly",
		"NEXT STEP: Run agnostic agent test",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("output missing %q\noutput:\n%s", expected, output)
		}
	}
}

func TestLoadCollaborationArtifactsSummarizesTypedCaptures(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)
	taskDir := filepath.Join(base, "projects", "myproject", "tasks", "agent-flow")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	files := map[string]string{
		"proposal.20260412T100000Z.aaaaaa.md": "# TASK PROPOSAL\n\n## Task\nagent-flow\n\n## Project\nmyproject\n\n## Capture Type\nproposal\n\n## Proposal\nUse --type for collaboration notes.\n",
		"request.20260412T110000Z.bbbbbb.md":  "# TASK REQUEST\n\n## Task\nagent-flow\n\n## Project\nmyproject\n\n## Capture Type\nrequest\n\n## Request\nClaude should review the parser.\n",
		"response.20260412T120000Z.cccccc.md": "# TASK RESPONSE\n\n## Task\nagent-flow\n\n## Project\nmyproject\n\n## Capture Type\nresponse\n\n## Recommendation\nKeep one command and use --type.\n",
		"decision.20260412T130000Z.dddddd.md": "# TASK DECISION\n\n## Task\nagent-flow\n\n## Project\nmyproject\n\n## Capture Type\ndecision\n\n## Decision\nAdopt typed captures.\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	artifacts, err := LoadCollaborationArtifacts("agent-flow", "myproject")
	if err != nil {
		t.Fatalf("LoadCollaborationArtifacts: %v", err)
	}
	if artifacts.ProposalCount != 1 || artifacts.RequestCount != 1 {
		t.Fatalf("unexpected counts: %+v", artifacts)
	}
	for _, want := range []string{
		"Use --type for collaboration notes.",
		"Claude should review the parser.",
		"Keep one command and use --type.",
		"Adopt typed captures.",
	} {
		got := strings.Join([]string{artifacts.LatestProposal, artifacts.LatestRequest, artifacts.LatestResponse, artifacts.LatestDecision}, "\n")
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q in %+v", want, artifacts)
		}
	}
}

func TestBuildContextPackageAddsCollaborationWithoutChangingState(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)
	taskDir := filepath.Join(base, "projects", "myproject", "tasks", "agent-flow")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "proposal.20260412T100000Z.aaaaaa.md"), []byte("# TASK PROPOSAL\n\n## Capture Type\nproposal\n\n## Proposal\nReview typed capture flow.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	snapshot := `# TASK SNAPSHOT

## Objective
Keep state clean

## Current State
- State remains authoritative

## Next Step
Run tests
`
	ctx := &ContextData{
		Profile:  "# Profile\n",
		Project:  "# PROJECT\n\n## Summary\nmy project\n",
		Snapshot: snapshot,
	}

	output := BuildContextPackage(ctx, "agent-flow", "myproject")
	for _, expected := range []string{
		"CURRENT STATE: State remains authoritative",
		"OPEN PROPOSALS: 1 (latest: Review typed capture flow.)",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("output missing %q\noutput:\n%s", expected, output)
		}
	}
}

func TestBuildContextPackageNoImplicitFocusWhenOnlyClosedTasksRemain(t *testing.T) {
	ctx := &ContextData{
		Profile:      "# Profile\n",
		Project:      "# PROJECT\n\n## Summary\nmy project\n",
		TaskContexts: map[string]*ContextData{},
	}

	output := BuildContextPackage(ctx, "", "myproject")
	if !strings.Contains(output, "CURRENT FOCUS: not yet defined") {
		t.Fatalf("expected no implicit focus\noutput:\n%s", output)
	}
}

func TestBuildCompactContextPackageWithTask(t *testing.T) {
	snapshot := `# TASK SNAPSHOT

## Objective
Build file explorer component for document navigation

## Current State
- routing ok
- FP table done
- upload pending

## Next Step
implement upload action in FP resource

## Active Issues
permission policy unclear
`
	ctx := &ContextData{
		Project:      "# PROJECT\n\n## Constraints\n- local-first storage\n- Spatie Shield RBAC\n",
		Snapshot:     snapshot,
		SnapshotName: "snapshot.20260325T143200Z.a3f2c1.md",
	}

	output := BuildCompactContextPackage(ctx, "explorer-ui", "metadocs")
	for _, expected := range []string{
		"PRJ:metadocs FOCUS:explorer-ui",
		"OBJ:Build file explorer component for document navigation",
		"STATE:routing ok | FP table done",
		"NEXT:implement upload action in FP resource",
		"ISSUES:permission policy unclear",
		"DECIDED:local-first storage | Spatie Shield RBAC",
		"SRC:snapshot.20260325T143200Z.a3f2c1.md",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("output missing %q\noutput:\n%s", expected, output)
		}
	}
}

func TestBuildCompactContextPackageOmitsEmptyOptionalFields(t *testing.T) {
	ctx := &ContextData{
		Project:  "# PROJECT\n\n## Summary\nmy project\n",
		Snapshot: "# TASK SNAPSHOT\n\n## Objective\n...\n",
	}

	output := BuildCompactContextPackage(ctx, "empty-task", "myproject")
	if strings.Contains(output, "OBJ:") || strings.Contains(output, "ISSUES:") || strings.Contains(output, "SRC:") {
		t.Fatalf("expected empty fields to be omitted\noutput:\n%s", output)
	}
	if !strings.Contains(output, "STATE:no snapshot yet") {
		t.Fatalf("expected fallback state for empty task\noutput:\n%s", output)
	}
}

func TestBuildCompactContextPackageListsProjectTasks(t *testing.T) {
	ctx := &ContextData{
		Project: "# PROJECT\n\n## Summary\nmy project\n",
		TaskContexts: map[string]*ContextData{
			"beta":  {},
			"alpha": {},
		},
	}

	output := BuildCompactContextPackage(ctx, "", "myproject")
	if !strings.Contains(output, "PRJ:myproject") {
		t.Fatalf("missing project header\noutput:\n%s", output)
	}
	if !strings.Contains(output, "TASKS:alpha | beta") {
		t.Fatalf("missing compact task list\noutput:\n%s", output)
	}
}
