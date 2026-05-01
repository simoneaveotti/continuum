package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"continuum/internal/events"
)

func withTempContinuum(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONTINUUM_PATH", dir)
	return dir
}

func writeTargets(t *testing.T, base string, targets []string) {
	t.Helper()
	content := "# test targets\n" + strings.Join(targets, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(base, "agent-targets.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadTargetFiles_Missing(t *testing.T) {
	withTempContinuum(t)
	_, err := loadTargetFiles()
	if err == nil {
		t.Error("expected error for missing agent-targets.txt, got nil")
	}
}

func TestLoadTargetFiles_Empty(t *testing.T) {
	base := withTempContinuum(t)
	os.WriteFile(filepath.Join(base, "agent-targets.txt"), []byte("# only comments\n"), 0o644)

	_, err := loadTargetFiles()
	if err == nil {
		t.Error("expected error for empty targets list, got nil")
	}
}

func TestLoadTargetFiles_Valid(t *testing.T) {
	base := withTempContinuum(t)
	writeTargets(t, base, []string{"AGENTS.md", "CLAUDE.md"})

	targets, err := loadTargetFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d: %v", len(targets), targets)
	}
}

func TestInstallToFile_AlreadyInstalled(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "AGENTS.md")

	content := "existing content\n" + MarkerStart + "\nbootstrap\n" + MarkerEnd + "\n"
	os.WriteFile(file, []byte(content), 0o644)

	// Change to the temp dir so installToFile finds the file
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	// Should skip (no force)
	err := installToFile("AGENTS.md", "myproject", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Content should be unchanged
	got, _ := os.ReadFile(file)
	if string(got) != content {
		t.Error("file should not have been modified when already installed and force=false")
	}
}

func TestInstallToFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	err := installToFile("NONEXISTENT.md", "myproject", false)
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestRemoveFromFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "AGENTS.md")

	content := "before\n\n" + MarkerStart + "\nbootstrap content\n" + MarkerEnd + "\n"
	os.WriteFile(file, []byte(content), 0o644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	if err := removeFromFile("AGENTS.md"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(file)
	if strings.Contains(string(got), MarkerStart) {
		t.Error("marker should have been removed")
	}
	if strings.Contains(string(got), "bootstrap content") {
		t.Error("bootstrap content should have been removed")
	}
}

func TestInstallAndRemoveEmitActivityEvents(t *testing.T) {
	base := withTempContinuum(t)
	writeTargets(t, base, []string{"AGENTS.md"})

	dir := t.TempDir()
	file := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(file, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	if err := Install("demo", false); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if err := Remove(); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	items, _, err := events.ReadFromOffset(0)
	if err != nil {
		t.Fatalf("ReadFromOffset: %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(items))
	}
	if items[len(items)-2].Type != "agent_install" || items[len(items)-1].Type != "agent_remove" {
		t.Fatalf("unexpected trailing events: %#v", items[len(items)-2:])
	}
}
