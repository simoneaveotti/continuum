package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func withTempContinuum(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONTINUUM_PATH", dir)
	return dir
}

func TestStart_CreatesFiles(t *testing.T) {
	base := withTempContinuum(t)

	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "my-project", "tasks", "my-task")
	if _, err := os.Stat(filepath.Join(taskDir, "notes.md")); err != nil {
		t.Errorf("expected notes.md to exist: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(taskDir, metadataFileName))
	if err != nil {
		t.Fatalf("expected %s to exist: %v", metadataFileName, err)
	}
	var meta taskMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if meta.Status != StatusActive {
		t.Fatalf("expected default status %q, got %q", StatusActive, meta.Status)
	}
}

// readLatestFileInDir reads the most recent file with the given prefix from dir.
func readLatestFileInDir(t *testing.T, dir, prefix string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatalf("no files with prefix %q found in %s", prefix, dir)
	}
	sort.Strings(names)
	data, err := os.ReadFile(filepath.Join(dir, names[len(names)-1]))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(data)
}

func TestStart_Idempotent(t *testing.T) {
	base := withTempContinuum(t)

	result, err := Start("my-task", "my-project")
	if err != nil {
		t.Fatalf("first Start() error: %v", err)
	}
	if result != StartCreated {
		t.Fatalf("first Start() result = %v, want %v", result, StartCreated)
	}
	activityPath := filepath.Join(base, "events", "activity.ndjson")
	before, err := os.ReadFile(activityPath)
	if err != nil {
		t.Fatalf("expected activity log after first Start(): %v", err)
	}

	result, err = Start("my-task", "my-project")
	if err != nil {
		t.Fatalf("second Start() error: %v", err)
	}
	if result != StartAlreadyActive {
		t.Fatalf("second Start() result = %v, want %v", result, StartAlreadyActive)
	}
	after, err := os.ReadFile(activityPath)
	if err != nil {
		t.Fatalf("expected activity log after second Start(): %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("second Start() should not append an activity event")
	}
}

func TestStart_ClosedTaskSuggestsReopen(t *testing.T) {
	withTempContinuum(t)

	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if _, err := SetStatus("my-task", "my-project", StatusClosed); err != nil {
		t.Fatalf("SetStatus() error: %v", err)
	}

	if _, err := Start("my-task", "my-project"); err == nil {
		t.Fatal("expected error for closed task")
	} else if !strings.Contains(err.Error(), "ctx task reopen my-task --project=my-project") {
		t.Fatalf("expected reopen suggestion, got %q", err)
	}
}

func TestSetStatus_WritesLifecycleEventTypes(t *testing.T) {
	base := withTempContinuum(t)

	if _, err := Start("my-task", "my-project"); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if _, err := SetStatus("my-task", "my-project", StatusClosed); err != nil {
		t.Fatalf("SetStatus(closed) error: %v", err)
	}
	if _, err := SetStatus("my-task", "my-project", StatusActive); err != nil {
		t.Fatalf("SetStatus(active) error: %v", err)
	}

	activityPath := filepath.Join(base, "events", "activity.ndjson")
	data, err := os.ReadFile(activityPath)
	if err != nil {
		t.Fatalf("ReadFile(activity): %v", err)
	}
	content := string(data)
	for _, want := range []string{`"type":"task_started"`, `"type":"task_closed"`, `"type":"task_reopened"`} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected activity log to contain %s\ncontent:\n%s", want, content)
		}
	}
}

func TestStart_EmptyTask(t *testing.T) {
	withTempContinuum(t)
	if _, err := Start("", "my-project"); err == nil {
		t.Error("expected error for empty task name, got nil")
	}
}

func TestStart_InvalidTaskName(t *testing.T) {
	withTempContinuum(t)
	if _, err := Start("../evil", "my-project"); err == nil {
		t.Fatal("expected error for invalid task name")
	}
}

func TestList_InvalidProjectName(t *testing.T) {
	withTempContinuum(t)
	if _, err := List("../evil"); err == nil {
		t.Fatal("expected error for invalid project name")
	}
}

func TestList_Empty(t *testing.T) {
	base := withTempContinuum(t)
	os.MkdirAll(filepath.Join(base, "projects", "my-project", "tasks"), 0o755)

	tasks, err := List("my-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestList(t *testing.T) {
	withTempContinuum(t)

	for _, name := range []string{"task-a", "task-b"} {
		if _, err := Start(name, "my-project"); err != nil {
			t.Fatal(err)
		}
	}

	tasks, err := List("my-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d: %v", len(tasks), tasks)
	}
}

func TestList_FiltersClosedTasksByDefault(t *testing.T) {
	withTempContinuum(t)

	if _, err := Start("active-task", "my-project"); err != nil {
		t.Fatal(err)
	}
	if _, err := Start("closed-task", "my-project"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetStatus("closed-task", "my-project", StatusClosed); err != nil {
		t.Fatal(err)
	}

	tasks, err := List("my-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 1 || tasks[0] != "active-task" {
		t.Fatalf("expected only active-task, got %v", tasks)
	}
}

func TestListWithStatus_All(t *testing.T) {
	withTempContinuum(t)

	if _, err := Start("active-task", "my-project"); err != nil {
		t.Fatal(err)
	}
	if _, err := Start("closed-task", "my-project"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetStatus("closed-task", "my-project", StatusClosed); err != nil {
		t.Fatal(err)
	}

	tasks, err := ListWithStatus("my-project", "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Name != "active-task" || tasks[0].Status != StatusActive {
		t.Fatalf("unexpected first task: %+v", tasks[0])
	}
	if tasks[1].Name != "closed-task" || tasks[1].Status != StatusClosed {
		t.Fatalf("unexpected second task: %+v", tasks[1])
	}
}
