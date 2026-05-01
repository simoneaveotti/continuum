package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"continuum/internal/filestore"
)

func TestArtifactListShowAndResolve(t *testing.T) {
	base := withTempContinuum(t)
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	taskDir := filepath.Join(base, "projects", "my-project", "tasks", "my-task")
	name := "proposal.20260413T120000Z.aaaaaa.md"
	content := "# TASK PROPOSAL\n\n## Proposal\nRead this artifact.\n"
	if err := os.WriteFile(filepath.Join(taskDir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	artifacts, err := ListArtifacts("my-task", "my-project", "all")
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d: %+v", len(artifacts), artifacts)
	}
	if artifacts[0].Name != name || artifacts[0].Type != filestore.ProposalCapture {
		t.Fatalf("unexpected artifact: %+v", artifacts[0])
	}

	got, err := ReadArtifact("my-task", "my-project", name)
	if err != nil {
		t.Fatalf("ReadArtifact: %v", err)
	}
	if !strings.Contains(got, "Read this artifact.") {
		t.Fatalf("unexpected artifact content: %q", got)
	}

	if err := ResolveArtifact("my-task", "my-project", name); err != nil {
		t.Fatalf("ResolveArtifact: %v", err)
	}
	artifacts, err = ListArtifacts("my-task", "my-project", "all")
	if err != nil {
		t.Fatalf("ListArtifacts after resolve: %v", err)
	}
	if len(artifacts) != 0 {
		t.Fatalf("expected resolved artifact to disappear from open list, got %+v", artifacts)
	}
	if _, err := os.Stat(filepath.Join(taskDir, "resolved", name)); err != nil {
		t.Fatalf("expected resolved artifact file: %v", err)
	}
}

func TestReadArtifactRejectsPathTraversal(t *testing.T) {
	withTempContinuum(t)
	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if _, err := ReadArtifact("my-task", "my-project", "../proposal.20260413T120000Z.aaaaaa.md"); err == nil {
		t.Fatal("expected path traversal artifact name to fail")
	}
}
