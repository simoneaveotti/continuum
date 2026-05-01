package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffUsesLatestTwoSnapshots(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	taskDir := filepath.Join(base, "projects", "demo", "tasks", "alpha")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	files := map[string]string{
		"snapshot.20260326T100000Z.aaaaaa.md": "# TASK SNAPSHOT\n\nold\n",
		"snapshot.20260326T110000Z.bbbbbb.md": "# TASK SNAPSHOT\n\nnew\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	diff, err := Diff("alpha", "demo", "", "")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(diff, "--- snapshot.20260326T100000Z.aaaaaa.md") {
		t.Fatalf("missing from header: %s", diff)
	}
	if !strings.Contains(diff, "+++ snapshot.20260326T110000Z.bbbbbb.md") {
		t.Fatalf("missing to header: %s", diff)
	}
	if !strings.Contains(diff, "-old") || !strings.Contains(diff, "+new") {
		t.Fatalf("missing diff body: %s", diff)
	}
}

func TestDiffSupportsExplicitSnapshotNames(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	taskDir := filepath.Join(base, "projects", "demo", "tasks", "alpha")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	for _, name := range []string{
		"snapshot.20260326T100000Z.aaaaaa.md",
		"snapshot.20260326T110000Z.bbbbbb.md",
		"snapshot.20260326T120000Z.cccccc.md",
	} {
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte(name+"\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	diff, err := Diff("alpha", "demo", "snapshot.20260326T100000Z.aaaaaa.md", "snapshot.20260326T120000Z.cccccc.md")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(diff, "-snapshot.20260326T100000Z.aaaaaa.md") {
		t.Fatalf("missing explicit from line: %s", diff)
	}
	if !strings.Contains(diff, "+snapshot.20260326T120000Z.cccccc.md") {
		t.Fatalf("missing explicit to line: %s", diff)
	}
}

func TestDiffRequiresTwoSnapshotsByDefault(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	taskDir := filepath.Join(base, "projects", "demo", "tasks", "alpha")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "snapshot.20260326T100000Z.aaaaaa.md"), []byte("only\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Diff("alpha", "demo", "", ""); err == nil {
		t.Fatal("expected error when only one snapshot exists")
	}
}

func TestDiffNoDifferences(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	taskDir := filepath.Join(base, "projects", "demo", "tasks", "alpha")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	for _, name := range []string{
		"snapshot.20260326T100000Z.aaaaaa.md",
		"snapshot.20260326T110000Z.bbbbbb.md",
	} {
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte("same\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	diff, err := Diff("alpha", "demo", "", "")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if diff != "No differences.\n" {
		t.Fatalf("unexpected diff output: %q", diff)
	}
}
