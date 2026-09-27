package task

import (
	"strings"
	"testing"
	"unicode/utf8"

	"continuum/internal/events"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestShortTSShowsClockWithSeconds(t *testing.T) {
	got := shortTS("2026-03-30T14:05:06Z")
	if got != "14:05:06" {
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

func TestProjectPickerFiltersAndSelects(t *testing.T) {
	m := newViewModel(100, 30)
	m.allProjects = []string{"alpha", "beta", "smoke-alpha"}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = next.(watchTUIModel)
	if !m.projectPicking {
		t.Fatal("p did not open the project picker")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("alp")})
	m = next.(watchTUIModel)
	if got := m.matchingProjects(); len(got) != 2 {
		t.Fatalf("matchingProjects() = %v, want two alpha projects", got)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(watchTUIModel)
	if m.projectSelected != 1 {
		t.Fatalf("projectSelected = %d, want 1", m.projectSelected)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(watchTUIModel)
	if m.projectPicking || len(m.projects) != 1 || m.projects[0] != "smoke-alpha" {
		t.Fatalf("picker did not select highlighted match: %+v", m)
	}
}

func TestProjectPickerAllowsJAndKInProjectNames(t *testing.T) {
	m := newViewModel(100, 30)
	m.allProjects = []string{"kappa", "jupiter"}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = next.(watchTUIModel)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = next.(watchTUIModel)
	if m.projectDraft != "k" {
		t.Fatalf("projectDraft = %q, want k", m.projectDraft)
	}
}

func TestModalKeyContract(t *testing.T) {
	m := newViewModel(100, 30)

	// Enter opens details from the list; Esc closes them without changing state.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = next.(watchTUIModel)
	if !m.info {
		t.Fatal("i must open details from the event list")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(watchTUIModel)
	if m.info {
		t.Fatal("esc must close details")
	}

	// Search owns input until explicitly confirmed or cancelled; i is literal text.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = next.(watchTUIModel)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = next.(watchTUIModel)
	if !m.searching || m.info || m.draft != "i" {
		t.Fatalf("search must retain input ownership: %+v", m)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(watchTUIModel)
	if m.searching {
		t.Fatal("esc must cancel search editing")
	}
}

func TestTruncateCellsAddsEllipsis(t *testing.T) {
	got := truncateCells("abcdef", 4)
	if got != "abc…" {
		t.Fatalf("truncateCells() = %q", got)
	}
}

// A byte-based truncation used to cut multi-byte runes in half, which broke
// the selection gutter into an invalid UTF-8 sequence.
func TestTruncateCellsNeverSplitsARune(t *testing.T) {
	got := truncateCells("▌marker", 1)
	if !utf8.ValidString(got) {
		t.Fatalf("truncateCells produced invalid UTF-8: %q", got)
	}
	if got := truncateCells("▌marker", 3); got != "▌m…" {
		t.Fatalf("truncateCells(\"▌marker\", 3) = %q, want %q", got, "▌m…")
	}
}

// The values below are the 26 distinct agent names found in a real
// 5492-event activity log, with their observed counts.
func TestNormalizedAgentFoldsRealSessionNames(t *testing.T) {
	folded := map[string]string{
		"codex":                              "codex",  // 3653
		"codex-root":                         "codex",  // 311
		"codex-docgraph":                     "codex",  // 1
		"codex-review-interval":              "codex",  // 1
		"codex-analyze-unmatched-authors":    "codex",  // 1
		"codex-unmatched-author-audit":       "codex",  // 3
		"codex_validate_reassociation_rules": "codex",  // 3
		"codex-laravel-messaging-review":     "codex",  // 1
		"codex-gcp-message-mvp":              "codex",  // 1
		"claude":                             "claude", // 733
		"claude-sonnet":                      "claude", // 7
		"gemini":                             "gemini", // 14
		"gemini-cli":                         "gemini", // 3
		"gpt":                                "gpt",    // 3
		"gpt-5":                              "gpt",    // 6
		"pccm-reviewer":                      "pccm",   // 3
		"spire-architect":                    "spire",  // 2
		"spire-ops":                          "spire",  // 1
		// Unqualified names are their own identity and must not be folded.
		"unknown":  "unknown", // 616
		"root":     "root",    // 66
		"opencode": "opencode",
		"pi":       "pi",
		"meyer":    "meyer",
		"metadocs": "metadocs",
		"":         "unknown",
		"   ":      "unknown",
	}
	for input, want := range folded {
		if got := normalizedAgent(input); got != want {
			t.Errorf("normalizedAgent(%q) = %q, want %q", input, got, want)
		}
	}
}

// The point of folding is not fewer distinct names, it is that a handful of
// identities cover the log. Weighted by the real counts, the top identities
// must account for essentially every event, otherwise the legend stays noise.
func TestNormalizedAgentConcentratesRealLogVolume(t *testing.T) {
	real := map[string]int{
		"codex": 3653, "codex-root": 311, "codex-unmatched-author-audit": 3,
		"codex_validate_reassociation_rules": 3, "codex-docgraph": 1,
		"codex-review-interval": 1, "codex-analyze-unmatched-authors": 1,
		"codex-laravel-messaging-review": 1, "codex-gcp-message-mvp": 1,
		"claude": 733, "claude-sonnet": 7,
		"gemini": 14, "gemini-cli": 3,
		"gpt": 3, "gpt-5": 6,
		"unknown": 616, "root": 66, "opencode": 54,
		"pi": 3, "meyer": 2, "metadocs": 2, "assistant": 2,
		"pccm-reviewer": 3, "spire-architect": 2, "spire-ops": 1,
	}

	buckets := map[string]int{}
	total := 0
	for name, count := range real {
		buckets[normalizedAgent(name)] += count
		total += count
	}

	// The five identities the legend can show must cover >99% of the log.
	leading := []string{"codex", "claude", "unknown", "root", "opencode"}
	covered := 0
	for _, name := range leading {
		covered += buckets[name]
	}
	share := float64(covered) / float64(total)
	if share < 0.99 {
		t.Fatalf("top identities cover only %.1f%% of %d events: %v", share*100, total, buckets)
	}

	// And the largest bucket must have absorbed the codex variants.
	if buckets["codex"] < 3900 {
		t.Fatalf("codex bucket is %d, expected the variants to be folded in: %v", buckets["codex"], buckets)
	}
}

func TestAgentQualifierReportsDroppedPart(t *testing.T) {
	cases := map[string]string{
		"codex-root":                         "root",
		"codex-docgraph":                     "docgraph",
		"codex_validate_reassociation_rules": "validate_reassociation_rules",
		"codex":                              "",
		"unknown":                            "",
		"":                                   "",
	}
	for input, want := range cases {
		if got := agentQualifier(input); got != want {
			t.Errorf("agentQualifier(%q) = %q, want %q", input, got, want)
		}
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

func TestWrapWordsBreaksOnWordBoundaries(t *testing.T) {
	got := wrapWords("a long detail that must not be truncated", 20)
	for _, line := range got {
		if lipgloss.Width(line) > 20 {
			t.Fatalf("line %q is %d cells, want <= 20", line, lipgloss.Width(line))
		}
	}
	joined := strings.Join(got, " ")
	if joined != "a long detail that must not be truncated" {
		t.Fatalf("wrapWords lost or reordered text: %q", joined)
	}
	// No word may be split across lines.
	words := map[string]bool{}
	for _, line := range got {
		for _, w := range strings.Fields(line) {
			words[w] = true
		}
	}
	for _, want := range []string{"detail", "truncated", "must"} {
		if !words[want] {
			t.Errorf("word %q was split across lines: %q", want, got)
		}
	}
}

// A single token wider than the pane must still be broken, not dropped.
func TestWrapWordsHardBreaksOverlongWord(t *testing.T) {
	got := wrapWords("projects/continuum/tasks/tui-polish/state.20260326T140506Z.a1b2c3.md", 20)
	if strings.Join(strings.Fields(strings.Join(got, "")), "") != "projects/continuum/tasks/tui-polish/state.20260326T140506Z.a1b2c3.md" {
		t.Fatalf("overlong word was not preserved: %q", got)
	}
	for _, line := range got {
		if lipgloss.Width(line) > 20 {
			t.Fatalf("line %q is %d cells, want <= 20", line, lipgloss.Width(line))
		}
	}
}
