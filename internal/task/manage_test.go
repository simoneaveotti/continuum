package task

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"continuum/internal/filestore"
	"continuum/internal/setup"
)

func TestSnapshotCleanKeepsLatest(t *testing.T) {
	base := withTempContinuum(t)

	if err := setup.Init("proj", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	if _, err := Start("my-task", "proj"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "proj", "tasks", "my-task")
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("snapshot.20260102T15040%dZ.%06x.md", i, i)
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte("snapshot"), 0o644); err != nil {
			t.Fatalf("WriteFile() error: %v", err)
		}
	}

	removed, err := SnapshotClean("my-task", "proj", 2)
	if err != nil {
		t.Fatalf("SnapshotClean() error: %v", err)
	}
	if removed != 3 {
		t.Fatalf("expected 3 snapshots removed, got %d", removed)
	}

	snapshots, err := filestore.AllSnapshots(taskDir)
	if err != nil {
		t.Fatalf("AllSnapshots() error: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots remaining, got %d", len(snapshots))
	}
}

func TestDeleteTaskRemovesDirectory(t *testing.T) {
	base := withTempContinuum(t)

	if err := setup.Init("proj", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	if _, err := Start("my-task", "proj"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "proj", "tasks", "my-task")
	notesPath := filepath.Join(taskDir, "notes.md")
	if err := os.WriteFile(notesPath, []byte("tracked notes"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	if out, err := exec.Command("git", "-C", base, "add", "--", filepath.Join("projects", "proj", "tasks", "my-task", "notes.md")).CombinedOutput(); err != nil {
		t.Fatalf("git add tracked note failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", base, "config", "user.email", "test@test.com").CombinedOutput(); err != nil {
		t.Fatalf("git config user.email failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", base, "config", "user.name", "Test").CombinedOutput(); err != nil {
		t.Fatalf("git config user.name failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", base, "commit", "-m", "track task note").CombinedOutput(); err != nil {
		t.Fatalf("git commit tracked note failed: %v\n%s", err, out)
	}

	if err := DeleteTask("my-task", "proj"); err != nil {
		t.Fatalf("DeleteTask() error: %v", err)
	}
	if _, err := os.Stat(taskDir); !os.IsNotExist(err) {
		t.Fatalf("expected task directory to be removed, got: %v", err)
	}

	out, err := exec.Command("git", "-C", base, "log", "--format=%s", "-1").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "delete(proj/my-task)") {
		t.Fatalf("unexpected commit message: %s", strings.TrimSpace(string(out)))
	}
}
