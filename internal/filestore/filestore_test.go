package filestore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewSnapshotName_Format(t *testing.T) {
	name := NewSnapshotName()
	if !strings.HasPrefix(name, "snapshot.") {
		t.Errorf("expected prefix snapshot., got %q", name)
	}
	if !strings.HasSuffix(name, ".md") {
		t.Errorf("expected suffix .md, got %q", name)
	}
	// Format: snapshot.<ts>.<hex6>.md — 4 parts when split by "."
	// snapshot . <ts> . <hex6> . md
	parts := strings.Split(name, ".")
	if len(parts) != 4 {
		t.Errorf("expected 4 dot-separated parts, got %d in %q", len(parts), name)
	}
}

func TestNewHandoffName_Format(t *testing.T) {
	name := NewHandoffName()
	if !strings.HasPrefix(name, "handoff.") {
		t.Errorf("expected prefix handoff., got %q", name)
	}
	if !strings.HasSuffix(name, ".md") {
		t.Errorf("expected suffix .md, got %q", name)
	}
}

func TestNewSnapshotName_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		name := NewSnapshotName()
		if seen[name] {
			t.Errorf("duplicate snapshot name: %q", name)
		}
		seen[name] = true
	}
}

func TestNewCaptureName_TypePrefixes(t *testing.T) {
	tests := map[CaptureType]string{
		StateCapture:    "snapshot.",
		ProposalCapture: "proposal.",
		RequestCapture:  "request.",
		ResponseCapture: "response.",
		DecisionCapture: "decision.",
	}
	for captureType, prefix := range tests {
		name := NewCaptureName(captureType)
		if !strings.HasPrefix(name, prefix) {
			t.Fatalf("NewCaptureName(%q) = %q, want prefix %q", captureType, name, prefix)
		}
		if !strings.HasSuffix(name, ".md") {
			t.Fatalf("NewCaptureName(%q) = %q, want .md suffix", captureType, name)
		}
	}
}

func TestValidateCaptureType(t *testing.T) {
	if got, err := ValidateCaptureType("proposal"); err != nil || got != ProposalCapture {
		t.Fatalf("ValidateCaptureType(proposal) = %q, %v", got, err)
	}
	if _, err := ValidateCaptureType("note"); err == nil {
		t.Fatal("expected invalid capture type error")
	}
}

func TestTimestamp_Format(t *testing.T) {
	ts := timestamp()
	// Should be parseable as UTC time in the expected layout
	_, err := time.ParseInLocation("20060102T150405Z", ts, time.UTC)
	if err != nil {
		t.Errorf("timestamp() = %q, cannot parse: %v", ts, err)
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := []byte("hello world")

	if err := AtomicWrite(path, content); err != nil {
		t.Fatalf("AtomicWrite() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("got %q, want %q", string(data), string(content))
	}

	// No temp files left behind
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp.") {
			t.Errorf("temp file not cleaned up: %s", e.Name())
		}
	}
}

func TestLatestSnapshot_Empty(t *testing.T) {
	dir := t.TempDir()
	path, name, err := LatestSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" || name != "" {
		t.Errorf("expected empty results, got path=%q name=%q", path, name)
	}
}

func TestLatestSnapshot_Single(t *testing.T) {
	dir := t.TempDir()
	fname := "snapshot.20260325T100000Z.abc123.md"
	os.WriteFile(filepath.Join(dir, fname), []byte("content"), 0o644)

	path, name, err := LatestSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != fname {
		t.Errorf("name = %q, want %q", name, fname)
	}
	if path != filepath.Join(dir, fname) {
		t.Errorf("path = %q, want %q", path, filepath.Join(dir, fname))
	}
}

func TestLatestSnapshot_ReturnsNewest(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"snapshot.20260325T100000Z.aaaaaa.md",
		"snapshot.20260325T120000Z.bbbbbb.md",
		"snapshot.20260325T080000Z.cccccc.md",
	}
	for _, f := range files {
		os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644)
	}

	_, name, err := LatestSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "snapshot.20260325T120000Z.bbbbbb.md"
	if name != want {
		t.Errorf("LatestSnapshot = %q, want %q", name, want)
	}
}

func TestLatestSnapshot_MissingDir(t *testing.T) {
	path, name, err := LatestSnapshot("/nonexistent/dir")
	if err != nil {
		t.Fatalf("unexpected error for missing dir: %v", err)
	}
	if path != "" || name != "" {
		t.Errorf("expected empty for missing dir, got path=%q name=%q", path, name)
	}
}

func TestAllSnapshots_Sorted(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"snapshot.20260325T120000Z.bbbbbb.md",
		"snapshot.20260325T080000Z.cccccc.md",
		"snapshot.20260325T100000Z.aaaaaa.md",
	}
	for _, f := range files {
		os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644)
	}

	paths, err := AllSnapshots(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}
	// Should be sorted: 08 < 10 < 12
	if !strings.Contains(paths[0], "080000") {
		t.Errorf("paths[0] = %q, expected the 08:00 file first", paths[0])
	}
	if !strings.Contains(paths[2], "120000") {
		t.Errorf("paths[2] = %q, expected the 12:00 file last", paths[2])
	}
}

func TestAllSnapshots_IgnoresHandoffFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "snapshot.20260325T100000Z.aaaaaa.md"), []byte("s"), 0o644)
	os.WriteFile(filepath.Join(dir, "handoff.20260325T100000Z.bbbbbb.md"), []byte("h"), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("n"), 0o644)

	paths, err := AllSnapshots(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 1 {
		t.Errorf("expected 1 snapshot, got %d: %v", len(paths), paths)
	}
}

func TestAllCapturesOfType_Sorted(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"proposal.20260325T120000Z.bbbbbb.md",
		"proposal.20260325T080000Z.cccccc.md",
		"snapshot.20260325T100000Z.aaaaaa.md",
	}
	for _, f := range files {
		os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644)
	}

	paths, err := AllCapturesOfType(dir, ProposalCapture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 proposal paths, got %d: %v", len(paths), paths)
	}
	if !strings.Contains(paths[0], "080000") || !strings.Contains(paths[1], "120000") {
		t.Fatalf("proposal paths not sorted: %v", paths)
	}
}
