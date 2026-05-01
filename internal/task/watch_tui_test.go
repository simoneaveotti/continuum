package task

import (
	"testing"

	"continuum/internal/events"
)

func TestShortTSIncludesDate(t *testing.T) {
	got := shortTS("2026-03-30T14:05:06Z")
	if got != "2026-03-30 14:05" {
		t.Fatalf("shortTS() = %q", got)
	}
}

func TestDetailTSIncludesSeconds(t *testing.T) {
	got := detailTS("2026-03-30T14:05:06Z")
	if got != "2026-03-30 14:05:06" {
		t.Fatalf("detailTS() = %q", got)
	}
}

func TestFilterEventsByProjects(t *testing.T) {
	items := []events.Event{
		{Project: "alpha", Type: "capture_saved"},
		{Project: "beta", Type: "task_started"},
		{Project: "", Type: "sync"},
	}

	got := filterEvents(items, []string{"alpha"})
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if got[0].Project != "alpha" || got[1].Project != "" {
		t.Fatalf("unexpected filtered events: %+v", got)
	}
}

func TestTrimAddsEllipsis(t *testing.T) {
	got := trim("abcdef", 4)
	if got != "abc…" {
		t.Fatalf("trim() = %q", got)
	}
}

func TestSortEventsNewestFirst(t *testing.T) {
	items := []events.Event{
		{Timestamp: "2026-03-26T10:00:00Z", Type: "older"},
		{Timestamp: "2026-03-26T10:00:02Z", Type: "newest"},
		{Timestamp: "2026-03-26T10:00:01Z", Type: "middle"},
	}

	got := sortEventsNewestFirst(items)

	if len(got) != 3 {
		t.Fatalf("expected 3 events, got %d", len(got))
	}
	if got[0].Type != "newest" || got[1].Type != "middle" || got[2].Type != "older" {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestFindEventIndex(t *testing.T) {
	items := []events.Event{
		{Timestamp: "2026-03-26T10:00:02Z", Type: "newest"},
		{Timestamp: "2026-03-26T10:00:01Z", Type: "middle"},
	}

	if idx := findEventIndex(items, items[1]); idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
}

func TestFitLinesRespectsHeight(t *testing.T) {
	got := fitLines("one\ntwo\nthree", 10, 2)
	if got != "one\ntwo" {
		t.Fatalf("unexpected fitted output: %q", got)
	}
}

func TestFitLinesTrimsWidth(t *testing.T) {
	got := fitLines("abcdef", 4, 1)
	if got != "abc…" {
		t.Fatalf("unexpected fitted output: %q", got)
	}
}

func TestWrapLinesWrapsLongPayloadLines(t *testing.T) {
	got := wrapLines("abcdefgh", 4)
	want := []string{"abcd", "efgh"}
	if len(got) != len(want) {
		t.Fatalf("wrapLines length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("wrapLines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFitWrappedLinesWindowRespectsOffset(t *testing.T) {
	lines := []string{"one", "two", "three", "four"}
	got := fitWrappedLinesWindow(lines, 2, 1)
	if len(got) != 2 || got[0] != "two" || got[1] != "three" {
		t.Fatalf("unexpected wrapped window: %#v", got)
	}
}

func TestProjectFilterIndex(t *testing.T) {
	if got := projectFilterIndex(nil, []string{"alpha", "beta"}); got != 0 {
		t.Fatalf("expected all-projects index 0, got %d", got)
	}
	if got := projectFilterIndex([]string{"beta"}, []string{"alpha", "beta"}); got != 2 {
		t.Fatalf("expected beta index 2, got %d", got)
	}
}

func TestCycleProjectFilter(t *testing.T) {
	m := watchTUIModel{
		allProjects: []string{"alpha", "beta"},
	}
	m.cycleProjectFilter()
	if len(m.projects) != 1 || m.projects[0] != "alpha" {
		t.Fatalf("expected alpha filter, got %#v", m.projects)
	}
	m.cycleProjectFilter()
	if len(m.projects) != 1 || m.projects[0] != "beta" {
		t.Fatalf("expected beta filter, got %#v", m.projects)
	}
	m.cycleProjectFilter()
	if len(m.projects) != 0 {
		t.Fatalf("expected all-projects filter, got %#v", m.projects)
	}
}
