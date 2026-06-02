package task

import (
	"strings"
	"testing"

	"continuum/internal/events"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestVisibleAgentOrderReturnsLastAgents(t *testing.T) {
	got := visibleAgentOrder([]string{"a", "b", "c", "d", "e", "f", "g"}, 5)
	want := []string{"c", "d", "e", "f", "g"}
	if len(got) != len(want) {
		t.Fatalf("visibleAgentOrder length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("visibleAgentOrder[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestVisibleAgentOrderCopiesWhenUnderLimit(t *testing.T) {
	source := []string{"a", "b"}
	got := visibleAgentOrder(source, 5)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected visibleAgentOrder result: %#v", got)
	}
	got[0] = "changed"
	if source[0] != "a" {
		t.Fatalf("visibleAgentOrder should return a copy, source mutated: %#v", source)
	}
}

func TestRenderLegendShowsHiddenAgentCount(t *testing.T) {
	m := watchTUIModel{
		agentOrder: []string{"a", "b", "c", "d", "e", "f", "g"},
		agentColors: map[string]lipgloss.Color{
			"a": lipgloss.Color("1"),
			"b": lipgloss.Color("2"),
			"c": lipgloss.Color("3"),
			"d": lipgloss.Color("4"),
			"e": lipgloss.Color("5"),
			"f": lipgloss.Color("6"),
			"g": lipgloss.Color("7"),
		},
	}

	got := m.renderLegend()
	for _, agent := range []string{"c", "d", "e", "f", "g"} {
		if !strings.Contains(got, agent) {
			t.Fatalf("renderLegend() missing visible agent %q: %q", agent, got)
		}
	}
	for _, agent := range []string{"a", "b"} {
		if strings.Contains(got, "● "+agent) {
			t.Fatalf("renderLegend() should hide agent %q: %q", agent, got)
		}
	}
	if !strings.Contains(got, "+2") {
		t.Fatalf("renderLegend() missing hidden count: %q", got)
	}
}

func TestRenderLegendShowsAllAgentsWhenExpanded(t *testing.T) {
	m := watchTUIModel{
		showAllAgents: true,
		agentOrder:    []string{"a", "b", "c", "d", "e", "f", "g"},
		agentColors: map[string]lipgloss.Color{
			"a": lipgloss.Color("1"),
			"b": lipgloss.Color("2"),
			"c": lipgloss.Color("3"),
			"d": lipgloss.Color("4"),
			"e": lipgloss.Color("5"),
			"f": lipgloss.Color("6"),
			"g": lipgloss.Color("7"),
		},
	}

	got := m.renderLegend()
	for _, agent := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		if !strings.Contains(got, "● "+agent) {
			t.Fatalf("renderLegend() missing expanded agent %q: %q", agent, got)
		}
	}
	if strings.Contains(got, "+") {
		t.Fatalf("renderLegend() should not show hidden count when expanded: %q", got)
	}
}

func TestToggleShowAllAgentsKey(t *testing.T) {
	m := watchTUIModel{}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	toggled, ok := next.(watchTUIModel)
	if !ok {
		t.Fatalf("Update() returned %T, want watchTUIModel", next)
	}
	if !toggled.showAllAgents {
		t.Fatalf("expected showAllAgents to be enabled")
	}

	next, _ = toggled.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	toggled, ok = next.(watchTUIModel)
	if !ok {
		t.Fatalf("Update() returned %T, want watchTUIModel", next)
	}
	if toggled.showAllAgents {
		t.Fatalf("expected showAllAgents to be disabled")
	}
}
