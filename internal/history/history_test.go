package history

import (
	"strings"
	"testing"
	"time"

	"continuum/internal/events"
)

func TestFilterTimelineEvents(t *testing.T) {
	items := []events.Event{
		{Timestamp: "2026-03-28T10:00:00Z", Project: "alpha", Task: "one", Type: "task_started"},
		{Timestamp: "2026-03-28T10:01:00Z", Project: "beta", Task: "two", Type: "capture"},
		{Timestamp: "2026-03-28T10:02:00Z", Project: "", Task: "", Type: "sync"},
	}

	got := filterTimelineEvents(items, "alpha", "", 0)
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if got[0].Project != "alpha" || got[1].Type != "sync" {
		t.Fatalf("unexpected filtered events: %+v", got)
	}
}

func TestDescribeEvent(t *testing.T) {
	item := events.Event{Type: "task_started", Detail: "task created"}
	if got := describeEvent(item); got != "started task: task created" {
		t.Fatalf("describeEvent() = %q", got)
	}
}

func TestHistoryHeader(t *testing.T) {
	got := historyHeader("History", "alpha", "one", 24*time.Hour, 3, "milestone(s)")
	if !strings.Contains(got, `project "alpha", task "one"`) {
		t.Fatalf("unexpected header: %q", got)
	}
	if !strings.Contains(got, "3 milestone(s)") {
		t.Fatalf("unexpected header count: %q", got)
	}
}

func TestRenderMilestone(t *testing.T) {
	item := milestone{
		Timestamp: "2026-03-28T10:00:00Z",
		Project:   "continuum",
		Task:      "history-command",
		Title:     "Clarified lifecycle rules",
		State:     "Capture threshold made stricter",
		Next:      "Validate on real usage",
		Source:    "snapshot.20260328T100000Z.abc123.md",
	}
	lines := renderMilestone(1, item)
	joined := strings.Join(lines, "\n")
	for _, needle := range []string{"continuum/history-command", "Clarified lifecycle rules", "State: Capture threshold made stricter", "Next: Validate on real usage"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected milestone output to contain %q, got:\n%s", needle, joined)
		}
	}
}

func TestCompressMilestones_DeduplicatesConsecutiveEquivalentItems(t *testing.T) {
	items := []milestone{
		{Timestamp: "2026-03-28T10:00:00Z", Project: "continuum", Task: "one", Kind: "snapshot", Title: "A", State: "B", Next: "C"},
		{Timestamp: "2026-03-28T10:01:00Z", Project: "continuum", Task: "one", Kind: "snapshot", Title: "A", State: "B", Next: "C"},
		{Timestamp: "2026-03-28T10:02:00Z", Project: "continuum", Task: "one", Kind: "snapshot", Title: "A2", State: "B", Next: "C"},
	}
	got := compressMilestones(items)
	if len(got) != 2 {
		t.Fatalf("expected 2 milestones after compression, got %d", len(got))
	}
}

func TestStyleHelpers_PlainWhenNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if got := styleTarget("continuum/task"); got != "continuum/task" {
		t.Fatalf("styleTarget() = %q, want unchanged text", got)
	}
}

func TestVisibleWidth_IgnoresANSI(t *testing.T) {
	value := "\x1b[38;5;110mhello\x1b[0m"
	if got := visibleWidth(value); got != 5 {
		t.Fatalf("visibleWidth() = %d, want 5", got)
	}
}
