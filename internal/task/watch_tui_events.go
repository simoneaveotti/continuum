// Event selection, filtering, sorting and agent identity for the watch TUI.
// This is the layer between the raw activity log and what the user sees.
package task

import (
	"continuum/internal/events"
	"github.com/charmbracelet/lipgloss"
	"slices"
	"strings"
)

// display returns the events the renderers should draw. With no active query
// this is the full buffer, so filtering costs a slice header, not a copy.
func (m watchTUIModel) display() []events.Event {
	if m.query == "" {
		return m.events
	}
	return m.matched
}

// clampSelection keeps the selection inside the displayed list.
func (m *watchTUIModel) clampSelection() {
	display := m.display()
	if len(display) == 0 {
		m.selected = 0
		m.payloadTop = 0
		return
	}
	if m.selected >= len(display) {
		m.selected = len(display) - 1
		m.payloadTop = 0
	}
	if m.selected < 0 {
		m.selected = 0
		m.payloadTop = 0
	}
}

// eventMatches reports whether an event matches the search query. The query is
// split on spaces and every term must appear somewhere in the event.
func eventMatches(item events.Event, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		item.Agent, item.Host, item.Project, item.Task,
		item.Type, item.Status, item.Detail, item.File,
	}, " "))
	for _, term := range terms {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func searchTerms(query string) []string {
	fields := strings.Fields(strings.ToLower(query))
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// recomputeMatch refreshes the filtered view for the current draft/query.

// recomputeMatch refreshes the filtered view for the current draft/query.

// recomputeMatch refreshes the filtered view for the current draft/query.
func (m *watchTUIModel) recomputeMatch() {
	terms := searchTerms(m.query)
	if len(terms) == 0 {
		m.matched = nil
		return
	}
	matched := make([]events.Event, 0, len(m.events))
	for _, item := range m.events {
		if eventMatches(item, terms) {
			matched = append(matched, item)
		}
	}
	m.matched = matched
}

// clampSelection keeps the selection inside the displayed list.

// trackSelection re-resolves the selected index after m.events changed. The
// event value alone is not a safe identity: Timestamp has second granularity,
// so two captures in the same second can be byte-identical and findEventIndex
// would jump the selection to the first of them.
func (m watchTUIModel) trackSelection(previous int) int {
	if previous < 0 || previous >= len(m.events) {
		return max(0, min(m.selected, len(m.events)-1))
	}
	if m.selected == previous && previous < len(m.events) {
		return previous
	}
	if idx := findEventIndex(m.events, m.events[previous]); idx >= 0 && idx != previous {
		// value moved: follow it
		m.selected = idx
	}
	return m.selected
}

// renderSectionHeader renders "- Title -───────────────" spanning width.

func (m *watchTUIModel) assignColors(items []events.Event) {
	if m.agentColors == nil {
		m.agentColors = map[string]lipgloss.Color{}
	}
	for _, item := range items {
		agent := normalizedAgent(item.Agent)
		if _, ok := m.agentColors[agent]; ok {
			continue
		}
		m.agentColors[agent] = agentPalette[len(m.agentOrder)%len(agentPalette)]
		m.agentOrder = append(m.agentOrder, agent)
	}
}

func (m *watchTUIModel) trimEvents() {
	if len(m.events) <= maxWatchEvents {
		return
	}
	// Remember how many events the buffer dropped so the header can say so:
	// a silently shrinking window reads as "nothing happened".
	m.dropped += len(m.events) - maxWatchEvents
	m.events = m.events[:maxWatchEvents]
	if m.selected >= len(m.events) {
		m.selected = max(0, len(m.events)-1)
	}
}

func (m *watchTUIModel) cycleProjectFilter() {
	if len(m.allProjects) == 0 {
		return
	}
	m.filterIndex = (m.filterIndex + 1) % (len(m.allProjects) + 1)
	if m.filterIndex == 0 {
		m.setProjectFilter(nil)
		return
	}
	m.setProjectFilter([]string{m.allProjects[m.filterIndex-1]})
}

func (m *watchTUIModel) setProjectFilter(projects []string) {
	m.projects = append([]string(nil), projects...)
	m.filterIndex = projectFilterIndex(m.projects, m.allProjects)
	m.events = sortEventsNewestFirst(filterEvents(readWatchEvents(), m.projects))
	m.selected = 0
	m.payloadTop = 0
	m.assignColors(m.events)
	m.trimEvents()
}

func projectFilterIndex(scope, allProjects []string) int {
	if len(scope) == 0 {
		return 0
	}
	if len(scope) == 1 {
		for i, project := range allProjects {
			if project == scope[0] {
				return i + 1
			}
		}
	}
	return 0
}

// readWatchEvents loads the events the watch buffer can hold, from the tail of
// the log rather than from the beginning.
func readWatchEvents() []events.Event {
	items, _, err := events.ReadTail(maxWatchEvents)
	if err != nil {
		return nil
	}
	return items
}

func filterEvents(items []events.Event, projects []string) []events.Event {
	if len(projects) == 0 {
		return items
	}
	var filtered []events.Event
	for _, item := range items {
		if item.Project == "" || slices.Contains(projects, item.Project) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// agentSeparators are the characters used by agent runtimes to qualify a
// session name, e.g. "codex-root" or "codex_validate_reassociation_rules".
// maxWatchEvents bounds the in-memory buffer. It is also the number of events
// read from the tail of the activity log at startup, so the two agree.
const maxWatchEvents = 200

// agentSeparators are the characters used by agent runtimes to qualify a
// session name, e.g. "codex-root" or "codex_validate_reassociation_rules".
const agentSeparators = "-_"

// normalizedAgent collapses an agent session name to the identity of the tool
// that produced it, so the legend groups the variants a single runtime emits.
//
// The real activity log contained 26 distinct values for what are really four
// tools: codex, codex-root, codex-docgraph, codex-review-interval,
// codex-analyze-unmatched-authors, codex_validate_reassociation_rules and
// friends. The legend capped at five entries, so it showed noise.
//
// Folding is display-only. The raw value stays in the event log and is still
// shown for the selected event, so no information is lost. Names that carry no
// qualifier ("unknown", "root", "pi") are returned unchanged: unattributed
// events are a real signal and must stay visible as their own bucket.
func normalizedAgent(agent string) string {
	trimmed := strings.TrimSpace(agent)
	if trimmed == "" {
		return "unknown"
	}
	if i := strings.IndexAny(trimmed, agentSeparators); i > 0 {
		return trimmed[:i]
	}
	return trimmed
}

// agentQualifier returns the part of the session name that normalizedAgent
// dropped, or "" when the name is already canonical.
func agentQualifier(agent string) string {
	trimmed := strings.TrimSpace(agent)
	if trimmed == "" {
		return ""
	}
	canonical := normalizedAgent(trimmed)
	if canonical == trimmed {
		return ""
	}
	return strings.Trim(trimmed[len(canonical):], agentSeparators)
}

func (m watchTUIModel) visibleAgents() []string {
	if m.showAllAgents {
		return append([]string(nil), m.agentOrder...)
	}
	return visibleAgentOrder(m.agentOrder, maxLegendAgents)
}

func visibleAgentOrder(agentOrder []string, limit int) []string {
	if limit <= 0 || len(agentOrder) <= limit {
		return append([]string(nil), agentOrder...)
	}
	return append([]string(nil), agentOrder[len(agentOrder)-limit:]...)
}

func sortEventsNewestFirst(items []events.Event) []events.Event {
	sorted := append([]events.Event(nil), items...)
	slices.SortStableFunc(sorted, func(a, b events.Event) int {
		switch {
		case a.Timestamp > b.Timestamp:
			return -1
		case a.Timestamp < b.Timestamp:
			return 1
		default:
			return 0
		}
	})
	return sorted
}

func findEventIndex(items []events.Event, target events.Event) int {
	for i, item := range items {
		if item == target {
			return i
		}
	}
	return 0
}

// resyncAfterRewrite rebuilds the watch buffer from the tail of a log that was
// rewritten underneath it, and records a notice so the user can tell the
// difference between a resync and a quiet period.
func (m *watchTUIModel) resyncAfterRewrite() {
	items, offset, err := events.ReadTail(maxWatchEvents)
	if err != nil {
		m.watchErr = err.Error()
		return
	}
	filtered := filterEvents(items, m.projects)
	m.assignColors(filtered)
	m.events = sortEventsNewestFirst(filtered)
	m.offset = offset
	m.dropped = 0
	m.selected = 0
	m.payloadTop = 0
	m.resolvePayload()
	m.watchErr = "activity log was rewritten, resynced"
}
