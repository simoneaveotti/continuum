package search

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSearchAcrossSnapshotsHandoffsAndArtifacts(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	taskDir := filepath.Join(base, "projects", "demo", "tasks", "alpha")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	files := map[string]string{
		"snapshot.20260326T100000Z.aaaaaa.md": "# TASK SNAPSHOT\n\nNeedle in snapshot\n",
		"handoff.20260326T110000Z.bbbbbb.md":  "# TASK HANDOFF\n\nneedle in handoff\n",
		"proposal.20260326T120000Z.cccccc.md": "# TASK PROPOSAL\n\nneedle in proposal\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	results, err := Search("needle", "", "", 0, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Kind != "proposal" {
		t.Fatalf("expected newest result to come from proposal, got %q", results[0].Kind)
	}
	if results[1].Kind != "handoff" {
		t.Fatalf("expected middle result to come from handoff, got %q", results[1].Kind)
	}
	if results[2].Kind != "snapshot" {
		t.Fatalf("expected oldest result to come from snapshot, got %q", results[2].Kind)
	}
}

func TestSearchProjectAndTaskFilters(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	write := func(project, task, file, content string) {
		t.Helper()
		dir := filepath.Join(base, "projects", project, "tasks", task)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	write("one", "alpha", "snapshot.20260326T100000Z.aaaaaa.md", "find me\n")
	write("two", "beta", "snapshot.20260326T100000Z.aaaaaa.md", "find me too\n")

	results, err := Search("find", "one", "alpha", 0, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Project != "one" || results[0].Task != "alpha" {
		t.Fatalf("unexpected result target: %+v", results[0])
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	if _, err := Search("", "", "", 0, 0); err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestSearchLimit(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	taskDir := filepath.Join(base, "projects", "demo", "tasks", "alpha")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	files := []string{
		"snapshot.20260326T100000Z.aaaaaa.md",
		"snapshot.20260326T110000Z.bbbbbb.md",
		"snapshot.20260326T120000Z.cccccc.md",
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte("needle\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	results, err := Search("needle", "", "", 2, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].File != "snapshot.20260326T120000Z.cccccc.md" {
		t.Fatalf("unexpected first result: %s", results[0].File)
	}
	if results[1].File != "snapshot.20260326T110000Z.bbbbbb.md" {
		t.Fatalf("unexpected second result: %s", results[1].File)
	}
}

func TestSearchSince(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	taskDir := filepath.Join(base, "projects", "demo", "tasks", "alpha")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	now := time.Now().UTC()
	recent := "snapshot." + now.Add(-2*time.Hour).Format("20060102T150405Z") + ".aaaaaa.md"
	old := "snapshot." + now.Add(-48*time.Hour).Format("20060102T150405Z") + ".bbbbbb.md"

	for _, name := range []string{recent, old} {
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte("needle\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	results, err := Search("needle", "", "", 0, 24*time.Hour)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].File != recent {
		t.Fatalf("unexpected result: %s", results[0].File)
	}
}
