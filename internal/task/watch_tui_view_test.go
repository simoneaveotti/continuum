package task

import (
	"fmt"
	"strings"
	"testing"

	"continuum/internal/events"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// newViewModel builds a model with deterministic dimensions and a few events.
func newViewModel(width, height int) watchTUIModel {
	items := []events.Event{
		{Timestamp: "2026-03-26T10:00:03Z", Agent: "claude", Host: "h1", Project: "p1", Task: "t1", Type: "capture", Status: "ok", Detail: "first"},
		{Timestamp: "2026-03-26T10:00:02Z", Agent: "codex", Host: "h1", Project: "p1", Task: "t2", Type: "capture", Status: "error", Detail: "second"},
		{Timestamp: "2026-03-26T10:00:01Z", Agent: "claude", Host: "h2", Project: "p2", Task: "t3", Type: "sync", Status: "ok", Detail: "third"},
	}
	m := watchTUIModel{
		width:       width,
		height:      height,
		interval:    2,
		events:      sortEventsNewestFirst(items),
		agentColors: map[string]lipgloss.Color{},
	}
	m.assignColors(m.events)
	return m
}

// View() must occupy the terminal exactly: one render too many lines makes the
// alt-screen scroll by a line on every frame and the whole layout drifts.
func TestViewFillsExactTerminalHeight(t *testing.T) {
	for _, tc := range []struct{ width, height int }{
		{120, 24}, {120, 30}, {100, 40}, {80, 50}, {60, 24}, {200, 60}, {40, 20},
	} {
		t.Run("", func(t *testing.T) {
			m := newViewModel(tc.width, tc.height)
			got := len(strings.Split(m.View(), "\n"))
			if got != tc.height {
				t.Fatalf("View() at %dx%d rendered %d lines, want exactly %d", tc.width, tc.height, got, tc.height)
			}
		})
	}
}

// No rendered line may exceed the terminal width.
func TestViewLinesFitTerminalWidth(t *testing.T) {
	for _, width := range []int{40, 60, 80, 100, 120, 200} {
		m := newViewModel(width, 30)
		for i, line := range strings.Split(m.View(), "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Fatalf("width %d: line %d is %d cells wide, want <= %d", width, i, w, width)
			}
		}
	}
}

// The vertical split must have a single source of truth. View and
// clampPayloadTop disagreed (60 vs 62 percent), which let the payload scroll
// past the last visible line.
func TestWatchLayoutSplitIsShared(t *testing.T) {
	for _, height := range []int{20, 24, 30, 40, 60} {
		_, topH, bottomH := watchLayout(100, height)
		if topH+bottomH != max(10, height-4) {
			t.Fatalf("height %d: topH(%d)+bottomH(%d) != bodyH(%d)", height, topH, bottomH, max(10, height-4))
		}
		if topH < 1 || bottomH < 2 {
			t.Fatalf("height %d: degenerate split topH=%d bottomH=%d", height, topH, bottomH)
		}
	}
}

// clampPayloadTop must not scroll past the content that is actually visible.
func TestClampPayloadTopStaysWithinVisibleContent(t *testing.T) {
	m := newViewModel(100, 30)
	m.events = []events.Event{{
		Timestamp: "2026-03-26T10:00:00Z",
		Type:      "capture",
		Status:    "ok",
		File:      "does-not-exist.md",
		Detail:    strings.Repeat("line\n", 500),
	}}
	m.payloadTop = 100000
	m.clampPayloadTop()

	_, _, bottomH := watchLayout(m.width, m.height)
	visible := max(1, bottomH-1)
	total := len(wrapLines(strings.Repeat("line\n", 500), max(1, m.width-2)))

	if maxTop := max(0, total-visible); m.payloadTop > maxTop {
		t.Fatalf("payloadTop %d exceeds max scroll %d (total=%d visible=%d)", m.payloadTop, maxTop, total, visible)
	}
}

// readEventPayload must not be called during View(): it reads from disk, and
// View runs on every tick, keypress and resize.
func TestViewDoesNotReadPayloadFromDisk(t *testing.T) {
	m := newViewModel(100, 30)
	m.events = []events.Event{{
		Timestamp: "2026-03-26T10:00:00Z",
		Type:      "capture",
		Status:    "ok",
		File:      "payload.md",
		Detail:    "fallback",
	}}
	// Same event in the model cache and on disk: if View() re-reads, the
	// cached value is ignored and the assertion below cannot tell. Instead we
	// assert the model exposes a resolved payload for the selection.
	first := m.View()
	m.payload = "SENTINEL"
	m.payloadValid = true
	m.payloadFor = m.selected
	second := m.View()
	if first == second {
		t.Fatalf("View() output did not change after altering payloadCache; payload is not model-driven")
	}
	if !strings.Contains(second, "SENTINEL") {
		t.Fatalf("View() ignored payloadCache; payload is still read from disk during render")
	}
}

// Duplicate events are indistinguishable because Timestamp has second
// granularity. Selection must still follow the row the user was on.
func TestFindEventIndexWithDuplicateEvents(t *testing.T) {
	dup := events.Event{Timestamp: "2026-03-26T10:00:00Z", Type: "capture", Status: "ok", Detail: "same"}
	items := []events.Event{
		{Timestamp: "2026-03-26T10:00:03Z", Type: "new"},
		dup,
		dup,
		{Timestamp: "2026-03-26T10:00:01Z", Type: "old"},
	}

	if idx := findEventIndex(items, dup); idx != 1 {
		t.Fatalf("findEventIndex on duplicates = %d, want 1 (first match)", idx)
	}

	// A selection-tracking helper must resolve the row the user was on, not
	// merely the first equal event.
	m := watchTUIModel{events: items, selected: 2}
	if got := m.trackSelection(2); got != 2 {
		t.Fatalf("trackSelection(2) = %d, want 2", got)
	}
}

func TestUpdateNavigationKeys(t *testing.T) {
	key := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	t.Run("j and k move selection", func(t *testing.T) {
		m := newViewModel(100, 30)
		next, _ := m.Update(key('j'))
		got := next.(watchTUIModel)
		if got.selected != 1 {
			t.Fatalf("j: selected = %d, want 1", got.selected)
		}
		next, _ = got.Update(key('k'))
		if back := next.(watchTUIModel); back.selected != 0 {
			t.Fatalf("k: selected = %d, want 0", back.selected)
		}
	})

	t.Run("selection clamps at both ends", func(t *testing.T) {
		m := newViewModel(100, 30)
		next, _ := m.Update(key('k'))
		if up := next.(watchTUIModel); up.selected != 0 {
			t.Fatalf("k at top: selected = %d, want 0", up.selected)
		}
		m.selected = len(m.events) - 1
		next, _ = m.Update(key('j'))
		if down := next.(watchTUIModel); down.selected != len(m.events)-1 {
			t.Fatalf("j at bottom: selected = %d, want %d", down.selected, len(m.events)-1)
		}
	})

	t.Run("g and G jump to bounds", func(t *testing.T) {
		m := newViewModel(100, 30)
		m.selected = 1
		next, _ := m.Update(key('G'))
		last := next.(watchTUIModel)
		if last.selected != len(m.events)-1 {
			t.Fatalf("G: selected = %d, want %d", last.selected, len(m.events)-1)
		}
		next, _ = last.Update(key('g'))
		if first := next.(watchTUIModel); first.selected != 0 {
			t.Fatalf("g: selected = %d, want 0", first.selected)
		}
	})

	t.Run("moving selection resets payload scroll", func(t *testing.T) {
		m := newViewModel(100, 30)
		m.payloadTop = 12
		next, _ := m.Update(key('j'))
		if got := next.(watchTUIModel); got.payloadTop != 0 {
			t.Fatalf("payloadTop = %d after j, want 0", got.payloadTop)
		}
	})
}

func TestUpdateWindowSizeStoresDimensions(t *testing.T) {
	m := watchTUIModel{}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 123, Height: 45})
	got := next.(watchTUIModel)
	if got.width != 123 || got.height != 45 {
		t.Fatalf("dimensions = %dx%d, want 123x45", got.width, got.height)
	}
}

func TestViewRendersEmptyState(t *testing.T) {
	m := watchTUIModel{width: 100, height: 30, agentColors: map[string]lipgloss.Color{}}
	out := m.View()
	if !strings.Contains(out, "no events") {
		t.Fatalf("expected an empty-state message, got:\n%s", out)
	}
	if got := len(strings.Split(out, "\n")); got != 30 {
		t.Fatalf("empty state rendered %d lines, want 30", got)
	}
}

func TestEventMatchesAllTerms(t *testing.T) {
	item := events.Event{Agent: "claude", Project: "continuum", Task: "tui", Type: "capture", Detail: "fix layout"}

	if !eventMatches(item, nil) {
		t.Fatal("empty terms must match everything")
	}
	if !eventMatches(item, searchTerms("claude tui")) {
		t.Fatal("expected both terms to match")
	}
	if eventMatches(item, searchTerms("claude gemini")) {
		t.Fatal("expected a missing term to reject the event")
	}
	if !eventMatches(item, searchTerms("CLAUDE")) {
		t.Fatal("search must be case-insensitive")
	}
}

func TestSearchNarrowsDisplay(t *testing.T) {
	m := newViewModel(100, 30)
	if got := len(m.display()); got != 3 {
		t.Fatalf("unfiltered display = %d, want 3", got)
	}

	m.query = "claude"
	m.recomputeMatch()
	if got := len(m.display()); got != 2 {
		t.Fatalf("filtered display = %d, want 2", got)
	}

	m.query = "nothing-matches-this"
	m.recomputeMatch()
	if got := len(m.display()); got != 0 {
		t.Fatalf("expected empty display, got %d", got)
	}
}

func TestUpdateSearchModeFlow(t *testing.T) {
	rune_ := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
	typeRune := func(m watchTUIModel, r rune) watchTUIModel {
		next, _ := m.Update(rune_(r))
		return next.(watchTUIModel)
	}

	m := newViewModel(100, 30)

	// "/" opens the prompt.
	next, _ := m.Update(rune_('/'))
	m = next.(watchTUIModel)
	if !m.searching {
		t.Fatal("/ must open the search prompt")
	}

	// Typing narrows the display incrementally.
	for _, r := range "codex" {
		m = typeRune(m, r)
	}
	if !strings.Contains(m.draft, "codex") {
		t.Fatalf("draft = %q, want it to contain codex", m.draft)
	}
	if got := len(m.display()); got != 1 {
		t.Fatalf("during search display = %d, want 1", got)
	}

	// Enter keeps the filter and closes the prompt.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(watchTUIModel)
	if m.searching {
		t.Fatal("enter must close the search prompt")
	}
	if m.query != "codex" {
		t.Fatalf("query = %q, want codex", m.query)
	}

	// esc clears a committed filter and restores the full list.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(watchTUIModel)
	if m.query != "" {
		t.Fatalf("esc must clear the query, got %q", m.query)
	}
	if got := len(m.display()); got != 3 {
		t.Fatalf("after esc display = %d, want 3", got)
	}
}

func TestUpdateSearchBackspace(t *testing.T) {
	rune_ := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
	m := newViewModel(100, 30)
	m.searching = true
	m.draft = "abc"
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = next.(watchTUIModel)
	if m.draft != "ab" {
		t.Fatalf("draft = %q, want ab", m.draft)
	}
	// Backspace on an empty draft must not panic or wrap.
	m.draft = ""
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got := next.(watchTUIModel).draft; got != "" {
		t.Fatalf("draft = %q, want empty", got)
	}
	_ = rune_
}

func TestUpdateHelpOverlayToggle(t *testing.T) {
	q := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
	m := newViewModel(100, 30)

	next, _ := m.Update(q('?'))
	m = next.(watchTUIModel)
	if !m.help {
		t.Fatal("? must open the help overlay")
	}
	// While help is open, navigation keys must not move the event selection.
	m.selected = 1
	next, _ = m.Update(q('j'))
	m = next.(watchTUIModel)
	if m.selected != 1 {
		t.Fatalf("selection moved to %d while help was open", m.selected)
	}
	next, _ = m.Update(q('?'))
	if next.(watchTUIModel).help {
		t.Fatal("? must close the help overlay")
	}
}

// The overlay must be height-exact too, or it reintroduces the scroll bug.
func TestHelpOverlayFitsExactHeight(t *testing.T) {
	for _, tc := range []struct{ width, height int }{{120, 24}, {100, 30}, {80, 40}, {60, 20}} {
		m := newViewModel(tc.width, tc.height)
		m.help = true
		if got := len(strings.Split(m.View(), "\n")); got != tc.height {
			t.Fatalf("help overlay at %dx%d rendered %d lines, want %d", tc.width, tc.height, got, tc.height)
		}
	}
}

func TestSearchPromptVisibleInFooter(t *testing.T) {
	m := newViewModel(100, 30)
	m.searching = true
	m.draft = "fix"
	if out := m.View(); !strings.Contains(out, "/fix") {
		t.Fatalf("expected the search prompt in the footer, got:\n%s", out)
	}
}

func TestHeaderSurfacesWatchErrorAndDropped(t *testing.T) {
	m := newViewModel(100, 30)
	m.watchErr = "cannot read activity stream"
	if out := m.View(); !strings.Contains(out, "cannot read activity stream") {
		t.Fatalf("expected the watch error in the header, got:\n%s", out)
	}

	m = newViewModel(100, 30)
	m.dropped = 42
	if out := m.View(); !strings.Contains(out, "42 dropped") {
		t.Fatalf("expected the dropped counter in the header, got:\n%s", out)
	}
}

func TestTrimEventsCountsDropped(t *testing.T) {
	m := newViewModel(100, 30)
	for len(m.events) <= 200 {
		m.events = append(m.events, events.Event{Timestamp: "2026-03-26T10:00:00Z", Type: "capture"})
	}
	before := len(m.events)
	m.trimEvents()
	if len(m.events) != 200 {
		t.Fatalf("events = %d, want 200", len(m.events))
	}
	if want := before - 200; m.dropped != want {
		t.Fatalf("dropped = %d, want %d", m.dropped, want)
	}
}

// Every help row must leave a gap between its key column and its description,
// including rows whose keys contain multi-cell glyphs like arrows.
func TestHelpRowsSeparateKeysFromDescription(t *testing.T) {
	keyW := 0
	for _, row := range helpRows {
		if w := lipgloss.Width(row[0]); w > keyW {
			keyW = w
		}
	}
	for _, row := range helpRows {
		line := renderHelpRow(row[0], row[1], 108, keyW)
		if lipgloss.Width(line) > 108 {
			t.Fatalf("row %q exceeds the panel width", row[0])
		}
		keysStr := strings.TrimRight(lipgloss.NewStyle().Render(fmt.Sprintf("  %-*s", keyW+1, row[0])), " ")
		if !strings.HasPrefix(line, keysStr+" ") {
			t.Fatalf("row %q has no gap before its description: %q", row[0], line)
		}
	}
}

// An applied search must be visible after Enter, otherwise a short list is
// indistinguishable from a quiet period.
func TestHeaderSurfacesActiveSearch(t *testing.T) {
	m := newViewModel(100, 30)
	m.searching = true
	m.draft = "codex"
	m.recomputeDraft()
	m.searching = false // user pressed Enter

	out := m.View()
	if !strings.Contains(out, "codex") {
		t.Fatalf("committed search query is not visible in the view:\n%s", out)
	}
	if !strings.Contains(out, `search "codex"`) {
		t.Fatalf("expected a search chip in the header, got:\n%s", out)
	}

	// And clearing it must remove the chip again.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	cleared := next.(watchTUIModel)
	if out := cleared.View(); strings.Contains(out, `search "`) {
		t.Fatalf("search chip still present after esc:\n%s", out)
	}
}

// The chip must not push the position indicator off a narrow terminal.
func TestSearchChipFitsNarrowTerminal(t *testing.T) {
	for _, width := range []int{40, 60, 80} {
		m := newViewModel(width, 24)
		m.query = "a-very-long-search-term-that-will-not-fit"
		m.recomputeMatch()
		out := m.View()
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Fatalf("width %d: header line %d is %d cells", width, i, w)
			}
		}
	}
}

// The CI has no TTY, so lipgloss strips styling and the ANSI path of
// truncateCells would otherwise never be exercised.
// The fields that do not fit in a row (storage path, host, folded session)
// must survive in the payload header, so removing the Details pane loses
// nothing.
func TestPayloadHeaderCarriesRowIncompatibleFields(t *testing.T) {
	m := newViewModel(100, 30)
	m.selected = 0
	m.events[0].Host = "mac126834"
	m.events[0].Agent = "codex-root"
	m.events[0].File = "projects/continuum/tasks/t1/state.20260326T100000Z.abc123.md"

	header := strings.Join(m.payloadHeaderLines(m.display()[m.selected]), "\n")
	for _, want := range []string{"Payload", "state.20260326T100000Z.abc123.md", "mac126834", "codex·root"} {
		if !strings.Contains(header, want) {
			t.Fatalf("payload header is missing %q: %q", want, header)
		}
	}
	// A canonical agent with no host or file must not grow empty segments.
	m.events[0].Agent = "codex"
	m.events[0].Host = ""
	m.events[0].File = ""
	got := strings.Join(m.payloadHeaderLines(m.display()[m.selected]), "\n")
	if strings.Contains(got, "  ·  ·  ") || strings.HasSuffix(got, "  ·  ") {
		t.Fatalf("payload header has an empty segment: %q", got)
	}
	for _, want := range []string{"Payload", detailTS(m.events[0].Timestamp), "ok"} {
		if !strings.Contains(got, want) {
			t.Fatalf("payload header lost %q: %q", want, got)
		}
	}
}

// The header must stay inside the pane at any width, since it now carries the
// longest strings in the view.
func TestPayloadHeaderFitsWidth(t *testing.T) {
	m := newViewModel(100, 30)
	m.selected = 0
	m.events[0].Host = "a-very-long-hostname.example.internal"
	m.events[0].Agent = "codex-an-extremely-long-session-name"
	m.events[0].File = "projects/continuum/tasks/t/a-very-long-snapshot-filename.20260326T100000Z.abcdef12.md"

	// The header block is at most two rows, so the payload pane never loses all
	// of its body to metadata.
	if n := len(m.payloadHeaderLines(m.display()[0])); n > 2 {
		t.Fatalf("payload header is %d lines, want at most 2", n)
	}

	// The renderer is responsible for truncating; the data is not width bound.
	for width := 20; width <= 200; width++ {
		for i, line := range strings.Split(m.renderPayloadSection(width, 10), "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Fatalf("width %d: payload line %d is %d cells", width, i, w)
			}
		}
	}
}

// The event table replaces the two-column layout: Detail belongs in a row
// (max 30 chars in the real log) and Status is "ok" in 99.8% of events, so it
// collapses to a single icon cell. That frees the width the Details pane was
// spending on five duplicated fields.
func TestEventsLayoutFreesWidthForDetail(t *testing.T) {
	l := layoutEvents(100)
	if !l.hasDetail || l.detail < 30 {
		t.Fatalf("detail column is %d cells, want >= 30 to hold real values: %+v", l.detail, l)
	}
	if l.status != 1 {
		t.Fatalf("status column is %d cells, want 1 (icon only): %+v", l.status, l)
	}
	if l.target < 12 {
		t.Fatalf("project/task column is %d cells, want >= 12: %+v", l.target, l)
	}
}

// Columns must be dropped or squeezed, never overlapped. This is the case
// that actually shipped broken once: the fixed columns summed to 22 and the
// target column still claimed its 12 minimum, so rows overflowed every
// terminal narrower than 34 columns.
func TestEventsLayoutNeverExceedsWidth(t *testing.T) {
	for width := 10; width <= 200; width++ {
		l := layoutEvents(width)
		if l.total() > width {
			t.Fatalf("width %d: columns total %d: %+v", width, l.total(), l)
		}
	}
}

func TestEventsLayoutDropsColumnsAsWidthShrinks(t *testing.T) {
	wide := layoutEvents(120)
	if !wide.hasDetail || !wide.hasType {
		t.Fatalf("120 columns should show every column: %+v", wide)
	}
	if layoutEvents(46).hasType {
		t.Fatalf("46 columns should have dropped Type: %+v", layoutEvents(46))
	}
	if layoutEvents(24).hasDetail {
		t.Fatalf("24 columns should have dropped Detail: %+v", layoutEvents(24))
	}
	tiny := layoutEvents(20)
	if tiny.time < 8 {
		t.Fatalf("time must survive at any width: %+v", tiny)
	}
	if tiny.agent < 5 {
		t.Fatalf("agent must stay readable at any width: %+v", tiny)
	}
}

// Every rendered row must fit its pane at every width, with the longest
// content the real log contains.
func TestEventRowsFitEveryWidth(t *testing.T) {
	long := strings.Repeat("W", 80)
	items := []events.Event{
		{Timestamp: "2026-09-26T10:04:11Z", Agent: "opencode", Host: "mac", Project: "a-long-project-name-here", Task: "a-long-task-name-here", Type: "project_initialized", Status: "error", Detail: long},
		{Timestamp: "2026-09-26T10:04:09Z", Agent: "codex", Host: "mac", Project: "p", Task: "t", Type: "sync", Status: "ok", Detail: "short"},
	}
	m := watchTUIModel{width: 100, height: 24, events: items, agentColors: map[string]lipgloss.Color{}}
	m.assignColors(m.events)
	m.selected = 0

	for width := 10; width <= 200; width++ {
		for i, line := range strings.Split(m.renderEventsSection(width, 12), "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Fatalf("width %d: row %d is %d cells: %q", width, i, w, line)
			}
		}
	}
}

// Detail must be visible in the row now that the Details pane is gone.
func TestEventRowShowsDetailAndType(t *testing.T) {
	m := newViewModel(120, 30)
	m.events[0].Detail = "moved payload reads out of View"
	m.events[0].Type = "capture"
	m.events[0].Status = "ok"
	rows := m.renderEventsSection(120, 12)
	if !strings.Contains(rows, "moved payload reads") {
		t.Fatalf("Detail is not rendered in the row:\n%s", rows)
	}
	if !strings.Contains(rows, "capture") {
		t.Fatalf("Type is not rendered in the row:\n%s", rows)
	}
	// Status is an icon cell now, so the word must not be spelled out.
	if strings.Contains(rows, "ok ") {
		t.Fatalf("Status should be icon-only:\n%s", rows)
	}
}

// The whole view must stay height-exact at every width, including the narrow
// ones where columns get dropped.
func TestViewFitsEveryWidthAndHeight(t *testing.T) {
	m := newViewModel(100, 24)
	for width := 10; width <= 200; width += 1 {
		for _, height := range []int{12, 16, 20, 24, 30, 40} {
			v := m
			v.width, v.height = width, height
			ls := strings.Split(v.View(), "\n")
			if len(ls) != height {
				t.Fatalf("%dx%d rendered %d lines", width, height, len(ls))
			}
			for i, line := range ls {
				if w := lipgloss.Width(line); w > width {
					t.Fatalf("%dx%d line %d is %d cells: %q", width, height, i, w, line)
				}
			}
		}
	}
}

// The agent legend used to wrap onto a second line on narrow terminals, which
// is the same overflow class as the footer bug: one extra row anywhere makes
// the view taller than the terminal.
func TestLegendNeverWraps(t *testing.T) {
	m := watchTUIModel{width: 100, height: 24, agentColors: map[string]lipgloss.Color{}}
	m.assignColors([]events.Event{
		{Agent: "codex"}, {Agent: "claude-sonnet"}, {Agent: "unknown"},
		{Agent: "opencode"}, {Agent: "gemini"}, {Agent: "gpt-5"},
	})
	legend := m.renderLegend()
	if strings.Contains(legend, "\n") {
		t.Fatalf("renderLegend() must return a single line, got %q", legend)
	}
	for width := 4; width <= 200; width++ {
		m.width = width
		if n := len(strings.Split(m.View(), "\n")); n != 24 {
			t.Fatalf("width %d: view rendered %d lines, want 24", width, n)
		}
	}
}

// Every fixed string in the view must be truncated, not wrapped: a wrapped
// line anywhere makes the view taller than the terminal. The header, footer,
// legend, section titles and the empty state have all wrapped at some point.
func TestFixedStringsNeverWrap(t *testing.T) {
	populated := newViewModel(100, 24)
	populated.assignColors([]events.Event{{Agent: "codex"}, {Agent: "claude-sonnet"}, {Agent: "unknown"}})
	empty := watchTUIModel{width: 100, height: 24, agentColors: map[string]lipgloss.Color{}}

	for name, m := range map[string]watchTUIModel{"populated": populated, "empty": empty} {
		for width := 4; width <= 200; width++ {
			v := m
			v.width = width
			ls := strings.Split(v.View(), "\n")
			if len(ls) != 24 {
				t.Fatalf("%s at width %d rendered %d lines, want 24", name, width, len(ls))
			}
			for i, line := range ls {
				if w := lipgloss.Width(line); w > width {
					t.Fatalf("%s at width %d line %d is %d cells: %q", name, width, i, w, line)
				}
			}
		}
	}
}

// WatchTUI used to read the entire activity log on startup: 1.4 MB and 5492
// events in the real log, parsed only to keep the newest 200.

// Dropping the Details pane must not lose information. The row dropped the
// date (it shows clock time only) and the status word (it shows an icon), so
// both have to reappear on the selected event's header.
func TestSelectedEventHeaderRestoresRowElidedFields(t *testing.T) {
	m := newViewModel(100, 30)
	m.selected = 0
	m.events[0].Timestamp = "2026-03-26T14:05:06Z"
	m.events[0].Status = "pending-review"
	m.events[0].Host = "pc126834"
	m.events[0].File = "projects/continuum/tasks/t1/state.20260326T140506Z.a1b2c3.md"

	header := strings.Join(m.payloadHeaderLines(m.display()[m.selected]), "\n")

	// The full timestamp, including the date the row drops.
	if !strings.Contains(header, "2026-03-26 14:05:06") {
		t.Fatalf("the full timestamp is no longer visible anywhere: %q", header)
	}
	// The status word, which the one cell icon column cannot carry.
	if !strings.Contains(header, "pending-review") {
		t.Fatalf("a non standard status is unreadable: %q", header)
	}
	// And the fields that were already relocated must still be there.
	for _, want := range []string{"pc126834", "state.20260326T140506Z.a1b2c3.md"} {
		if !strings.Contains(header, want) {
			t.Fatalf("lost %q when the details pane was removed: %q", want, header)
		}
	}
}

// The details pane was removed because it duplicated the table row, but the
// extended view of one event is still wanted. It must be reachable on demand
// and must show values in full, not truncated.
func TestEventInfoOverlayShowsEveryFieldInFull(t *testing.T) {
	m := newViewModel(100, 30)
	m.selected = 0
	m.events[0] = events.Event{
		Timestamp: "2026-03-26T14:05:06Z",
		Agent:     "codex-root",
		Host:      "pc126834",
		Project:   "continuum",
		Task:      "tui-polish",
		Type:      "capture",
		Status:    "pending-review",
		Detail:    strings.Repeat("a long detail that must not be truncated ", 4),
		File:      "projects/continuum/tasks/tui-polish/state.20260326T140506Z.a1b2c3.md",
	}
	item := m.display()[m.selected]

	m.info = true
	out := m.View()
	// Wrapping breaks long values across rows, so compare with whitespace
	// normalised: the point is that nothing is dropped, not where it breaks.
	flat := strings.Join(strings.Fields(out), " ")

	for _, want := range []string{
		"2026-03-26T14:05:06Z", "14:05:06", "codex-root", "pc126834",
		"continuum", "tui-polish", "capture", "pending-review",
		// The full storage path, not just the basename the header shows.
		"projects/continuum/tasks/tui-polish/state.20260326T140506Z.a1b2c3.md",
		// And the detail in full, however long it is.
		strings.Join(strings.Fields(item.Detail), " "),
	} {
		needle := strings.Join(strings.Fields(want), " ")
		if !strings.Contains(flat, needle) {
			t.Errorf("extended view is missing %q", truncateForMessage(needle))
		}
	}
	// It is a modal: it owns the whole terminal.
	if n := len(strings.Split(out, "\n")); n != 30 {
		t.Errorf("extended view rendered %d lines, want the full 30", n)
	}
}

func truncateForMessage(s string) string {
	if len(s) > 40 {
		return s[:37] + "..."
	}
	return s
}

func TestEventInfoOverlayOpensAndCloses(t *testing.T) {
	info := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
	m := newViewModel(100, 30)

	next, _ := m.Update(info('i'))
	m = next.(watchTUIModel)
	if !m.info {
		t.Fatal("i must open the extended view")
	}
	// Navigation keys must not move the selection while the modal is open.
	m.selected = 1
	next, _ = m.Update(info('j'))
	if next.(watchTUIModel).selected != 1 {
		t.Fatal("selection moved while the extended view was open")
	}
	for _, key := range []rune{'i', 'q'} {
		m = next.(watchTUIModel)
		m.info = true
		next, _ = m.Update(info(key))
		if next.(watchTUIModel).info {
			t.Fatalf("%q must close the extended view", key)
		}
	}
	// The empty state must not panic.
	empty := watchTUIModel{width: 100, height: 30, agentColors: map[string]lipgloss.Color{}, info: true}
	if n := len(strings.Split(empty.View(), "\n")); n != 30 {
		t.Fatalf("empty extended view rendered %d lines, want 30", n)
	}
}

// projectSelected could go negative: min(len(matches)-1, ...) is -1 when nothing
// matches, and Enter then indexed matches[-1].
func TestProjectPickerNeverIndexesOutOfRange(t *testing.T) {
	typeMsg := func(m watchTUIModel, s string) watchTUIModel {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
		return next.(watchTUIModel)
	}
	m := newViewModel(100, 30)
	m.allProjects = []string{"alpha", "beta", "smoke-alpha"}
	m = typeMsg(m, "p")

	// No match at all: arrowing down must not produce a negative index.
	m = typeMsg(m, "zzz")
	if len(m.matchingProjects()) != 0 {
		t.Fatalf("expected no matches, got %v", m.matchingProjects())
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(watchTUIModel)
	if m.projectSelected < 0 {
		t.Fatalf("projectSelected = %d with no matches, want >= 0", m.projectSelected)
	}

	// Widen the match set again and select: this used to panic.
	for range 3 {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = next.(watchTUIModel)
	}
	if m.projectSelected < 0 || m.projectSelected >= len(m.matchingProjects()) {
		t.Fatalf("projectSelected = %d out of range for %d matches", m.projectSelected, len(m.matchingProjects()))
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // must not panic
	m = next.(watchTUIModel)
	if m.projectPicking {
		t.Fatal("enter must close the picker")
	}
}

// The picker renders project names, help text and an empty state: every one of
// them has to be truncated, or a narrow terminal scrolls the view.
func TestProjectPickerFitsEveryWidth(t *testing.T) {
	long := strings.Repeat("long-project-name-", 12)
	m := watchTUIModel{
		width: 100, height: 30,
		allProjects:    []string{long, "alpha", "beta", strings.Repeat("z", 90)},
		agentColors:    map[string]lipgloss.Color{},
		events:         nil,
		projectPicking: true,
		projectDraft:   "zzz", // forces the "no matching projects" path too
	}
	for _, draft := range []string{"", "zzz", "alpha"} {
		v := m
		v.projectDraft = draft
		for width := 4; width <= 200; width++ {
			v.width = width
			ls := strings.Split(v.View(), "\n")
			if len(ls) != 30 {
				t.Fatalf("draft %q at width %d rendered %d lines, want 30", draft, width, len(ls))
			}
			for i, line := range ls {
				if w := lipgloss.Width(line); w > width {
					t.Fatalf("draft %q at width %d line %d is %d cells: %q", draft, width, i, w, line)
				}
			}
		}
	}
}

// The height sweep must cover every modal the view can enter, otherwise a new
// one silently escapes the width contract.
func TestWidthContractCoversEveryModal(t *testing.T) {
	base := newViewModel(100, 24)
	base.allProjects = []string{strings.Repeat("p", 60), "alpha"}
	modals := map[string]func(watchTUIModel) watchTUIModel{
		"table":  func(m watchTUIModel) watchTUIModel { return m },
		"help":   func(m watchTUIModel) watchTUIModel { m.help = true; return m },
		"info":   func(m watchTUIModel) watchTUIModel { m.info = true; return m },
		"picker": func(m watchTUIModel) watchTUIModel { m.projectPicking = true; return m },
		"search": func(m watchTUIModel) watchTUIModel { m.searching = true; m.draft = "some query"; return m },
		"empty":  func(m watchTUIModel) watchTUIModel { m.events = nil; m.matched = nil; return m },
	}
	for name, enter := range modals {
		for width := 4; width <= 160; width++ {
			v := enter(base)
			v.width, v.height = width, 24
			ls := strings.Split(v.View(), "\n")
			if len(ls) != 24 {
				t.Fatalf("%s at width %d rendered %d lines, want 24", name, width, len(ls))
			}
			for i, line := range ls {
				if w := lipgloss.Width(line); w > width {
					t.Fatalf("%s at width %d line %d is %d cells: %q", name, width, i, w, line)
				}
			}
		}
	}
}

func TestTruncateCellsIgnoresANSIEscapes(t *testing.T) {
	styled := "\x1b[38;5;179m" + `search "a-fairly-long-query-value"` + "\x1b[0m"
	got := truncateCells(styled, 20)
	if w := lipgloss.Width(got); w > 20 {
		t.Fatalf("truncated to %d cells, want <= 20: %q", w, got)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatal("styling was dropped instead of preserved")
	}
	// A styled string must keep the same visible budget as a plain one.
	plain := `search "a-fairly-long-query-value"`
	if a, b := lipgloss.Width(truncateCells(plain, 20)), lipgloss.Width(truncateCells(styled, 20)); a != b {
		t.Fatalf("styled width %d != plain width %d", b, a)
	}
}

func TestTruncateCellsRuneAware(t *testing.T) {
	full := "  q quit  ·  j/k events"
	if got := truncateCells(full, 100); got != full {
		t.Fatalf("short string was modified: %q", got)
	}
	// A byte-based trim would over-count these multi-byte glyphs and cut early.
	got := truncateCells(full, 10)
	if w := lipgloss.Width(got); w > 10 {
		t.Fatalf("truncated to %d cells, want <= 10: %q", w, got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected an ellipsis, got %q", got)
	}
	if got := truncateCells(full, 0); got != "" {
		t.Fatalf("width 0 must produce empty string, got %q", got)
	}
}
