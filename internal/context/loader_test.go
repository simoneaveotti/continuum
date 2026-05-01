package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"continuum/internal/setup"
)

func TestLoadMissingTaskReturnsClearError(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	if err := setup.InitSession(false); err != nil {
		t.Fatalf("InitSession: %v", err)
	}
	if err := setup.Init("demo", false); err != nil {
		t.Fatalf("Init project: %v", err)
	}

	_, err := Load("missing-task", "demo")
	if err == nil {
		t.Fatal("expected missing task error")
	}
	if !strings.Contains(err.Error(), `task "missing-task" not found in project "demo"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadReadsExistingTask(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	if err := setup.InitSession(false); err != nil {
		t.Fatalf("InitSession: %v", err)
	}
	if err := setup.Init("demo", false); err != nil {
		t.Fatalf("Init project: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "demo", "tasks", "existing-task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	snapshotPath := filepath.Join(taskDir, "snapshot.20260326T000000Z.test.md")
	if err := os.WriteFile(snapshotPath, []byte("## Objective\n- test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ctxData, err := Load("existing-task", "demo")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ctxData.Snapshot == "" {
		t.Fatal("expected snapshot content")
	}
}
