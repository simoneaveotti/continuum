package task

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"continuum/internal/events"
	"continuum/internal/setup"

	"github.com/charmbracelet/lipgloss"
)

func TestReadTailReturnsOnlyTheNewestEvents(t *testing.T) {
	log := writeActivityLog(t, 500)

	items, offset, err := events.ReadTail(200)
	if err != nil {
		t.Fatalf("ReadTail() error: %v", err)
	}
	if len(items) != 200 {
		t.Fatalf("ReadTail(200) returned %d events, want 200", len(items))
	}
	// Oldest and newest must be the real ends of the log.
	if items[0].Detail != "event 300" {
		t.Fatalf("oldest returned is %q, want %q", items[0].Detail, "event 300")
	}
	last := items[len(items)-1]
	if last.Detail != "event 499" {
		t.Fatalf("newest returned is %q, want %q", last.Detail, "event 499")
	}
	// The offset must be the end of the file so incremental reads continue.
	info, err := os.Stat(log)
	if err != nil {
		t.Fatal(err)
	}
	if offset != info.Size() {
		t.Fatalf("offset = %d, want the file size %d", offset, info.Size())
	}
	// Reading from that offset must yield nothing new.
	more, _, err := events.ReadFromOffset(offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(more) != 0 {
		t.Fatalf("expected no events after the tail offset, got %d", len(more))
	}
}

func TestReadTailSmallerThanLogAndThanLimit(t *testing.T) {
	writeActivityLog(t, 10)
	items, _, err := events.ReadTail(200)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 10 {
		t.Fatalf("ReadTail on a 10 event log returned %d, want all 10", len(items))
	}

	items, _, err = events.ReadTail(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[2].Detail != "event 9" {
		t.Fatalf("ReadTail(3) = %d events ending %q", len(items), items[len(items)-1].Detail)
	}
}

func TestReadTailOnMissingAndEmptyLog(t *testing.T) {
	t.Setenv("CONTINUUM_PATH", t.TempDir())
	items, offset, err := events.ReadTail(200)
	if err != nil {
		t.Fatalf("ReadTail on a missing log must not fail: %v", err)
	}
	if items != nil || offset != 0 {
		t.Fatalf("got %v/%d, want nil/0", items, offset)
	}
}

// A log whose last line was not flushed must not yield a truncated record.
func TestReadTailIgnoresPartialTrailingLine(t *testing.T) {
	writeActivityLog(t, 5)
	path := events.ActivityPath()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"timestamp":"2026-03-26T10:00:05Z","type":"cap`); err != nil {
		t.Fatal(err)
	}
	f.Close()

	items, _, err := events.ReadTail(200)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 5 {
		t.Fatalf("got %d events, want the 5 complete ones", len(items))
	}
	for _, item := range items {
		if item.Detail == "" {
			t.Fatalf("partial line leaked into results: %+v", item)
		}
	}
}

// The tail read must scale with the limit, not with the size of the log.
func TestReadTailDoesNotScaleWithLogSize(t *testing.T) {
	writeActivityLog(t, 40000)

	_, _, err := events.ReadTail(200)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			readTailForBench(200)
		}
	})
	// Reading 200 of 40000 events should stay in the millisecond range; a full
	// parse of the log would be orders of magnitude slower.
	if elapsed.NsPerOp() > 50*1000*1000 {
		t.Fatalf("ReadTail(200) over 40000 events took %s, expected a tail read", elapsed)
	}
	t.Logf("ReadTail(200) over a 40000 event log: %s", elapsed)
}

func readTailForBench(n int) {
	_, _, _ = events.ReadTail(n)
}

// writeActivityLog writes count events with sequential details and returns the
// path.
func writeActivityLog(t *testing.T, count int) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONTINUUM_PATH", dir)

	path := filepath.Join(dir, "events", "activity.ndjson")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for i := 0; i < count; i++ {
		ts := time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Second)
		b.WriteString(fmt.Sprintf(
			`{"timestamp":%q,"agent":"codex","host":"h","project":"p","task":"t","type":"capture","status":"ok","detail":"event %d"}`+"\n",
			ts.Format(time.RFC3339), i))
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Cycling the project filter re-reads the log, and it used to do so from the
// beginning every time.
func TestSetProjectFilterReadsTailNotWholeLog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONTINUUM_PATH", dir)
	path := filepath.Join(dir, "events", "activity.ndjson")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	// 3000 events for project "keep", then 5 for "other": the tail read must
	// not be dominated by the older ones when filtering to "other".
	for i := 0; i < 3000; i++ {
		b.WriteString(`{"timestamp":"2026-03-26T10:00:00Z","agent":"codex","host":"h","project":"keep","task":"t","type":"capture","status":"ok","detail":"old"}` + "\n")
	}
	for i := 0; i < 5; i++ {
		b.WriteString(`{"timestamp":"2026-03-26T10:00:00Z","agent":"codex","host":"h","project":"other","task":"t","type":"capture","status":"ok","detail":"new"}` + "\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	m := watchTUIModel{allProjects: []string{"other"}, agentColors: map[string]lipgloss.Color{}}
	m.setProjectFilter([]string{"other"})
	if len(m.events) != 5 {
		t.Fatalf("filtered to %d events, want the 5 in the tail", len(m.events))
	}
	for _, item := range m.events {
		if item.Project != "other" {
			t.Fatalf("filter leaked a %q event", item.Project)
		}
	}
}

// readEventPayload joined the storage path with event.File and read it without
// checking containment. event.File comes from the log, so a log entry is
// enough to make the TUI read an arbitrary file.
func TestReadEventPayloadRejectsEscapingPaths(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONTINUUM_PATH", dir)

	secret := filepath.Join(dir, "..", "outside-secret.md")
	if err := os.WriteFile(secret, []byte("TOP SECRET"), 0o644); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(dir, "projects", "p", "tasks", "t", "state.md")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inside, []byte("legit payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	escapes := []string{
		"../outside-secret.md",
		"../../outside-secret.md",
		"projects/../../outside-secret.md",
		"..",
		"../",
	}
	for _, file := range escapes {
		got := readEventPayload(events.Event{File: file})
		if got != "" {
			t.Errorf("readEventPayload(%q) = %q, want it refused", file, got)
		}
	}

	// A legitimate relative path must still resolve.
	got := readEventPayload(events.Event{File: "projects/p/tasks/t/state.md"})
	if got != "legit payload" {
		t.Fatalf("legitimate payload = %q, want it read", got)
	}
}

// ctx repair --activity rewrites the log. A watch holding a byte offset from
// before the rewrite would seek into the middle of a line, and every event from
// there on would fail to parse and be dropped without a word.
func TestReadFromOffsetDetectsRewrittenLog(t *testing.T) {
	writeActivityLog(t, 10)
	_, offset, err := events.ReadFromOffset(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := events.ReadFromOffset(offset); err != nil {
		t.Fatalf("a valid offset must read cleanly, got %v", err)
	}

	// Simulate the repair shrinking and rewriting the log.
	path := events.ActivityPath()
	shrunk := `{"timestamp":"2026-03-26T09:00:00Z","agent":"codex","host":"h","project":"p","task":"t","type":"clean","status":"ok","detail":"rewritten"}` + "\n"
	if err := os.WriteFile(path, []byte(shrunk), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err = events.ReadFromOffset(offset)
	if !errors.Is(err, events.ErrLogRewritten) {
		t.Fatalf("after a rewrite, ReadFromOffset returned %v, want ErrLogRewritten", err)
	}
}

// A truncated log must be detected too, not read past the end.
func TestReadFromOffsetDetectsTruncatedLog(t *testing.T) {
	writeActivityLog(t, 10)
	_, offset, err := events.ReadFromOffset(0)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(events.ActivityPath(), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := events.ReadFromOffset(offset); !errors.Is(err, events.ErrLogRewritten) {
		t.Fatalf("after truncation, ReadFromOffset returned %v, want ErrLogRewritten", err)
	}
}

// The watch must recover from a rewrite by re-reading the tail, and must say
// so: a silent resync looks exactly like an agent that stopped working.
func TestWatchResyncsAfterLogRewrite(t *testing.T) {
	writeActivityLog(t, 5)
	m := newViewModel(100, 30)
	_, offset, err := events.ReadFromOffset(0)
	if err != nil {
		t.Fatal(err)
	}
	m.offset = offset
	m.events = nil

	// Rewrite the log underneath the running watch.
	rewritten := `{"timestamp":"2026-03-26T09:00:00Z","agent":"claude","host":"h","project":"p","task":"t","type":"clean","status":"ok","detail":"rewritten"}` + "\n"
	if err := os.WriteFile(events.ActivityPath(), []byte(rewritten), 0o644); err != nil {
		t.Fatal(err)
	}

	next, _ := m.Update(tuiTickMsg{})
	got := next.(watchTUIModel)
	if len(got.events) != 1 || got.events[0].Detail != "rewritten" {
		t.Fatalf("watch did not resync from the rewritten log: %+v", got.events)
	}
	if got.watchErr == "" {
		t.Fatal("the rewrite must be surfaced, not swallowed")
	}
	if !strings.Contains(got.View(), "resynced") {
		t.Fatalf("the view does not mention the resync:\n%s", got.View())
	}
}

func TestWatchResyncsAfterActivityRepair(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	t.Setenv("CONTINUUM_PATH", dir)
	if err := setup.InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}

	lines := []string{
		`{"timestamp":"2026-03-26T10:00:02Z","agent":"codex","host":"h","project":"p","task":"t","type":"capture","status":"ok","detail":"later"}`,
		`{"timestamp":"2026-03-26T10:00:01Z","agent":"claude","host":"h","project":"p","task":"t","type":"capture","status":"ok","detail":"earlier"}`,
		`{"timestamp":"2026-03-26T10:00:02Z","agent":"codex","host":"h","project":"p","task":"t","type":"capture","status":"ok","detail":"later"}`,
		`<<<<<<< HEAD`,
	}
	if err := os.MkdirAll(filepath.Dir(events.ActivityPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(events.ActivityPath(), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, offset, err := events.ReadFromOffset(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setup.RepairActivityLog(); err != nil {
		t.Fatalf("RepairActivityLog() error: %v", err)
	}

	m := newViewModel(100, 30)
	m.offset = offset
	m.events = nil
	next, _ := m.Update(tuiTickMsg{})
	got := next.(watchTUIModel)
	if len(got.events) != 2 || got.events[0].Detail != "later" || got.events[1].Detail != "earlier" {
		t.Fatalf("watch did not resync from the repaired log: %+v", got.events)
	}
	if !strings.Contains(got.watchErr, "resynced") {
		t.Fatalf("watch did not report repair resync: %q", got.watchErr)
	}
}
