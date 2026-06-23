package events

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTempDir(t *testing.T) string {
	t.Helper()
	path := t.TempDir()
	t.Setenv("CONTINUUM_PATH", path)
	return path
}

func TestAppendCreatesFile(t *testing.T) {
	base := setupTempDir(t)

	if err := Append("myproject", "mytask", "capture", "ok", "test event"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	path := filepath.Join(base, activityRelPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("activity file not created at %s", path)
	}
}

func TestAppendAndReadBack(t *testing.T) {
	setupTempDir(t)

	if err := Append("p1", "t1", "test_event", "ok", "detail text"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	items, offset, err := ReadFromOffset(0)
	if err != nil {
		t.Fatalf("ReadFromOffset: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 event, got %d", len(items))
	}
	if items[0].Project != "p1" {
		t.Errorf("Project = %q, want %q", items[0].Project, "p1")
	}
	if items[0].Task != "t1" {
		t.Errorf("Task = %q, want %q", items[0].Task, "t1")
	}
	if items[0].Type != "test_event" {
		t.Errorf("Type = %q, want %q", items[0].Type, "test_event")
	}
	if items[0].Status != "ok" {
		t.Errorf("Status = %q, want %q", items[0].Status, "ok")
	}
	if items[0].Detail != "detail text" {
		t.Errorf("Detail = %q, want %q", items[0].Detail, "detail text")
	}
	if offset == 0 {
		t.Errorf("expected non-zero offset after read")
	}
}

func TestAppendMultipleEvents(t *testing.T) {
	setupTempDir(t)

	if err := Append("p1", "t1", "start", "ok", "first"); err != nil {
		t.Fatalf("Append first: %v", err)
	}
	if err := Append("p1", "t1", "stop", "ok", "second"); err != nil {
		t.Fatalf("Append second: %v", err)
	}

	items, _, err := ReadFromOffset(0)
	if err != nil {
		t.Fatalf("ReadFromOffset: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 events, got %d", len(items))
	}
	if items[0].Type != "start" || items[1].Type != "stop" {
		t.Errorf("unexpected event order: %+v", items)
	}
}

func TestReadFromOffsetSkipsProcessed(t *testing.T) {
	setupTempDir(t)

	if err := Append("p1", "t1", "a", "ok", "first"); err != nil {
		t.Fatalf("Append first: %v", err)
	}
	if err := Append("p1", "t1", "b", "ok", "second"); err != nil {
		t.Fatalf("Append second: %v", err)
	}

	items1, offset, err := ReadFromOffset(0)
	if err != nil {
		t.Fatalf("ReadFromOffset: %v", err)
	}
	if len(items1) != 2 {
		t.Fatalf("expected 2 events from start, got %d", len(items1))
	}

	items2, offset2, err := ReadFromOffset(offset)
	if err != nil {
		t.Fatalf("ReadFromOffset with offset: %v", err)
	}
	if len(items2) != 0 {
		t.Fatalf("expected 0 new events, got %d", len(items2))
	}
	if offset2 != offset {
		t.Errorf("offset changed from %d to %d", offset, offset2)
	}

	if err := Append("p1", "t1", "c", "ok", "third"); err != nil {
		t.Fatalf("Append third: %v", err)
	}

	items3, _, err := ReadFromOffset(offset2)
	if err != nil {
		t.Fatalf("ReadFromOffset after append: %v", err)
	}
	if len(items3) != 1 {
		t.Fatalf("expected 1 new event, got %d", len(items3))
	}
	if items3[0].Type != "c" {
		t.Errorf("expected event type 'c', got %q", items3[0].Type)
	}
}

func TestAppendWithFile(t *testing.T) {
	setupTempDir(t)

	if err := AppendWithFile("p1", "t1", "export", "ok", "exported task", "task.zip"); err != nil {
		t.Fatalf("AppendWithFile: %v", err)
	}

	items, _, err := ReadFromOffset(0)
	if err != nil {
		t.Fatalf("ReadFromOffset: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 event, got %d", len(items))
	}
	if items[0].File != "task.zip" {
		t.Errorf("File = %q, want %q", items[0].File, "task.zip")
	}
}

func TestReadFromOffsetNoFile(t *testing.T) {
	setupTempDir(t)

	items, offset, err := ReadFromOffset(0)
	if err != nil {
		t.Fatalf("ReadFromOffset on missing file: %v", err)
	}
	if items != nil {
		t.Errorf("expected nil items, got %v", items)
	}
	if offset != 0 {
		t.Errorf("expected offset 0, got %d", offset)
	}
}

func TestActivityRelPath(t *testing.T) {
	if !strings.HasSuffix(ActivityRelPath(), "activity.ndjson") {
		t.Errorf("unexpected rel path: %s", ActivityRelPath())
	}
}

func TestActivityPathUsesEnv(t *testing.T) {
	base := setupTempDir(t)
	path := ActivityPath()
	expected := filepath.Join(base, activityRelPath)
	if path != expected {
		t.Errorf("ActivityPath() = %q, want %q", path, expected)
	}
}
