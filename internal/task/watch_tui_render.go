// Rendering for the watch TUI. Every function here turns model state into
// styled, width-exact lines; none of them mutate the model or read from disk.
package task

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"continuum/internal/events"
)

func (m watchTUIModel) renderHeaderBar() string {
	pos := "0/0"
	if display := m.display(); len(display) > 0 {
		pos = fmt.Sprintf("%d/%d", m.selected+1, len(display))
	}
	scopeLabel := "all projects"
	if len(m.projects) > 0 {
		scopeLabel = "project: " + strings.Join(m.projects, ",")
	}
	left := fmt.Sprintf("continuum watch  ·  %s  ·  %d events  ·  %d agents  ·  ↻ %s",
		scopeLabel, len(m.display()), len(m.agentOrder), m.interval)
	if m.dropped > 0 {
		left += fmt.Sprintf("  ·  %d dropped", m.dropped)
	}
	// The search chip goes last on purpose: the header truncates from the
	// right, so a long query gives up its space before the counts do.
	if m.query != "" {
		left += "  ·  " + tuiRunningStyle.Render(fmt.Sprintf("search %q", m.query))
	}
	if m.watchErr != "" {
		left = tuiErrStyle.Render("⚠ " + truncateCells(m.watchErr, max(0, m.width-4)))
	}
	// right-align pos; Padding(0,1) adds 2 chars so available = width-2
	// tuiBarStyle pads by one cell each side, so the content budget is
	// width-2. Truncating to width here would overflow the padding and wrap.
	available := m.width - 2
	if available < 4 {
		return truncateCells(left, max(0, available))
	}
	left = truncateCells(left, max(0, available-len(pos)-1))
	gap := available - lipgloss.Width(left) - lipgloss.Width(pos)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + pos
}

func (m watchTUIModel) renderLegend() string {
	if len(m.agentOrder) == 0 {
		return tuiMuted.Render("  no agents")
	}
	visibleAgents := m.visibleAgents()
	parts := make([]string, 0, len(visibleAgents))
	for _, agent := range visibleAgents {
		agentStyle := lipgloss.NewStyle().Foreground(m.agentColors[agent])
		parts = append(parts, agentStyle.Render("● "+agent))
	}
	if hiddenCount := len(m.agentOrder) - len(visibleAgents); hiddenCount > 0 {
		parts = append(parts, tuiMuted.Render(fmt.Sprintf("+%d", hiddenCount)))
	}
	return "  " + strings.Join(parts, "   ")
}

// eventsLayout is the column plan for the event table. It is computed rather
// than hardcoded so narrow terminals drop columns instead of overlapping them.
type eventsLayout struct {
	gutter    int
	time      int
	agent     int
	kind      int
	status    int
	target    int
	detail    int
	hasType   bool
	hasDetail bool
}

// total is the number of cells the row occupies, gaps included. The renderer
// emits three fixed gaps (after time, after agent, after status) plus one
// before each optional column, and this must stay in step with it.
func (l eventsLayout) total() int {
	n := l.gutter + l.time + l.agent + l.status + 3
	if l.hasType {
		n += 1 + l.kind
	}
	if l.hasDetail {
		n += 1 + l.detail
	}
	return n + l.target
}

// layoutEvents plans the table for a given width.
//
// The widths come from the real activity log: agent identities fold to at
// most ~10 cells, Type is 12, Detail peaks at 30 characters, and Status is
// "ok" in 99.8% of events so it only needs the icon.
func layoutEvents(width int) eventsLayout {
	l := eventsLayout{gutter: 1, time: 8, agent: 10, status: 1}
	// Budget everything against the width instead of computing the remainder
	// from fixed sizes, so no combination of columns can overflow the row.
	budget := width - (l.gutter + l.status + 3)

	l.time = max(0, min(l.time, budget))
	budget -= l.time
	l.agent = max(0, min(l.agent, budget))
	budget -= l.agent

	const typeW = 12
	if budget >= typeW+15 {
		l.hasType = true
		l.kind = typeW
		budget -= 1 + typeW
	}

	l.target = max(0, min(30, budget*45/100))
	budget -= l.target
	if budget >= 11 {
		l.hasDetail = true
		l.detail = min(36, budget-1)
		budget -= 1 + l.detail
	}
	if budget > 0 {
		l.target += budget
	}
	return l
}

func (m watchTUIModel) renderEventsSection(width, height int) string {
	titleLine := renderSectionHeader("Events", width)
	l := layoutEvents(width)
	timeW, agentW, typeW, statusW := l.time, l.agent, l.kind, l.status
	targetW := l.target

	headerCells := []string{
		renderCell("", l.gutter, tuiHeaderRow),
		renderCell("Time", timeW, tuiHeaderRow),
		renderCell("", 1, tuiHeaderRow),
		renderCell("Agent", agentW, tuiHeaderRow),
		renderCell("", 1, tuiHeaderRow),
	}
	if l.hasType {
		headerCells = append(headerCells,
			renderCell("Type", typeW, tuiHeaderRow),
			renderCell("", 1, tuiHeaderRow),
		)
	}
	headerCells = append(headerCells,
		renderCell("", statusW, tuiHeaderRow),
		renderCell("", 1, tuiHeaderRow),
		renderCell("Project/Task", targetW, tuiHeaderRow),
	)
	if l.hasDetail {
		headerCells = append(headerCells, renderCell("Detail", l.detail, tuiHeaderRow))
	}
	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, headerCells...)
	sepLine := tuiSepStyle.Width(width).Render(strings.Repeat("─", width))

	fixed := []string{titleLine, headerLine, sepLine}
	rowsH := max(1, height-len(fixed))

	var eventRows []string
	display := m.display()
	if len(display) == 0 {
		empty := "  no events yet"
		if m.query != "" {
			empty = fmt.Sprintf("  no events match %q", m.query)
		}
		eventRows = []string{tuiMuted.Width(width).Render(truncateCells(empty, width))}
	} else {
		windowStart := m.selected - rowsH/2
		if windowStart < 0 {
			windowStart = 0
		}
		if windowStart+rowsH > len(display) {
			windowStart = max(0, len(display)-rowsH)
		}
		windowEnd := min(len(display), windowStart+rowsH)

		for i := windowStart; i < windowEnd; i++ {
			item := display[i]
			agent := normalizedAgent(item.Agent)
			baseStyle := tuiDefaultCell
			if i == m.selected {
				baseStyle = tuiSelectedRow
			}
			// The gutter is a redundant selection cue: background colour alone
			// is invisible on low-contrast or transparent terminals.
			gutter := " "
			if i == m.selected {
				gutter = tuiSelectionStyle.Render("▌")
			}
			cells := []string{renderCell(gutter, l.gutter, baseStyle)}
			if timeW > 0 {
				cells = append(cells,
					renderCell(shortTS(item.Timestamp), timeW, baseStyle),
					renderCell("", 1, baseStyle),
				)
			}
			if agentW > 0 {
				cells = append(cells,
					renderCell(agent, agentW, baseStyle.Copy().Foreground(m.agentColors[agent]).Bold(true)),
					renderCell("", 1, baseStyle),
				)
			}
			if l.hasType {
				cells = append(cells,
					renderCell(item.Type, typeW, baseStyle),
					renderCell("", 1, baseStyle),
				)
			}
			cells = append(cells,
				renderCell(statusIcon(item.Status), statusW, mergeStyles(baseStyle, statusTextStyle(item.Status))),
				renderCell("", 1, baseStyle),
				renderCell(eventTarget(item), targetW, baseStyle),
			)
			if l.hasDetail {
				cells = append(cells, renderCell(item.Detail, l.detail, baseStyle))
			}
			eventRows = append(eventRows, lipgloss.JoinHorizontal(lipgloss.Left, cells...))
		}
	}

	lines := append(fixed, eventRows...)
	return fitRenderedLines(lines, width, height)
}

// payloadHeaderLines is the full identity of the selected event, as a block
// rather than one line.
//
// The table row deliberately elides precision so a long list stays scannable:
// it shows "15:04:05" instead of the date, and a one cell status icon instead
// of the status word. This block is where that precision comes back, along with
// the storage path, host and session that have no room in a column at all.
//
// It is two lines on purpose. Crammed onto one, the tail gets truncated on a
// normal width terminal, which is how the session qualifier went missing once
// already.
func (m watchTUIModel) payloadHeaderLines(item events.Event) []string {
	primary := []string{"Payload"}
	if ts := detailTS(item.Timestamp); ts != "" {
		primary = append(primary, ts)
	}
	if item.Status != "" {
		// Only decorate a status we have an icon for, so an unrecognised status
		// does not render as an empty "·" segment.
		if icon := statusIcon(item.Status); icon != "·" {
			primary = append(primary, icon+" "+item.Status)
		} else {
			primary = append(primary, item.Status)
		}
	}

	var secondary []string
	if item.File != "" {
		secondary = append(secondary, filepath.Base(item.File))
	}
	if item.Host != "" {
		secondary = append(secondary, item.Host)
	}
	if qualifier := agentQualifier(item.Agent); qualifier != "" {
		secondary = append(secondary, normalizedAgent(item.Agent)+"·"+qualifier)
	}
	if len(secondary) == 0 {
		return []string{strings.Join(primary, "  ·  ")}
	}
	return []string{strings.Join(primary, "  ·  "), "  " + strings.Join(secondary, "  ·  ")}
}

func (m watchTUIModel) renderPayloadSection(width, height int) string {
	display := m.display()
	header := []string{"Payload"}
	if len(display) > 0 {
		header = m.payloadHeaderLines(display[m.selected])
	}

	var content string
	if len(display) == 0 {
		content = "no payload"
	} else {
		item := display[m.selected]
		content = strings.TrimSpace(m.payloadForSelection())
		if content == "" {
			content = item.Detail
		}
		if content == "" {
			content = "no payload"
		}
	}

	// Scroll indicator appended to title when scrolled
	wrappedLines := wrapLines(content, max(1, width-2))
	totalLines := len(wrappedLines)
	scrollIndicator := ""
	if m.payloadTop > 0 {
		scrollIndicator = fmt.Sprintf("  ↑↓ line %d/%d", m.payloadTop+1, totalLines)
	}
	header[0] = renderSectionHeader(header[0]+scrollIndicator, width)
	// The continuation lines carry no rule, but they still have to fit.
	for i := 1; i < len(header); i++ {
		header[i] = tuiLabel.Width(width).MaxWidth(width).Render(truncateCells(header[i], width))
	}

	// Reserve a row for every header line, not just the first.
	contentH := max(1, height-len(header))
	rawLines := fitWrappedLinesWindow(wrappedLines, contentH, m.payloadTop)

	// Render content lines with dark payload background
	rendered := make([]string, 0, height)
	rendered = append(rendered, header...)
	for _, line := range rawLines {
		rendered = append(rendered, tuiPayloadLine.Width(width).MaxWidth(width).Render("  "+line))
		if len(rendered) == height {
			break
		}
	}
	for len(rendered) < height {
		rendered = append(rendered, tuiPayloadLine.Width(width).Render(""))
	}
	return strings.Join(rendered, "\n")
}

// renderSectionHeader renders "- Title -───────────────" spanning width.

// renderSectionHeader renders "- Title -───────────────" spanning width.
// renderSectionHeader renders "- Title -───────────────" spanning width. The
// title is truncated first: a long payload filename would otherwise push the
// whole line past the terminal width.
func renderSectionHeader(title string, width int) string {
	const chrome = len("-  -") + 1 // "- " + " -" plus the separator cell
	if width <= chrome {
		return tuiSepStyle.Render(truncateCells(title, max(0, width)))
	}
	titleStr := tuiSectionTitle.Render("- " + truncateCells(title, width-chrome) + " -")
	titleW := lipgloss.Width(titleStr)
	lineW := max(0, width-titleW-1)
	return titleStr + tuiSepStyle.Render(strings.Repeat("─", lineW))
}

func statusIcon(status string) string {
	switch strings.ToLower(status) {
	case "ok", "success", "done", "completed":
		return "✓"
	case "error", "failed", "failure":
		return "✗"
	case "running", "in_progress":
		return "⟳"
	default:
		return "·"
	}
}

func renderCell(value string, width int, style lipgloss.Style) string {
	return style.Width(width).MaxWidth(width).Render(truncateCells(value, width))
}

func mergeStyles(base, overlay lipgloss.Style) lipgloss.Style {
	if fg := overlay.GetForeground(); fg != nil {
		base = base.Foreground(fg)
	}
	if bg := overlay.GetBackground(); bg != nil {
		base = base.Background(bg)
	}
	if overlay.GetBold() {
		base = base.Bold(true)
	}
	return base
}

func statusTextStyle(status string) lipgloss.Style {
	switch strings.ToLower(status) {
	case "ok", "success", "done", "completed":
		return tuiOkStyle
	case "error", "failed", "failure":
		return tuiErrStyle
	case "running", "in_progress":
		return tuiRunningStyle
	default:
		return tuiDefaultStatus
	}
}

// helpRows is the content of the ? overlay.
var helpRows = [][2]string{
	{"q / ctrl+c", "quit"},
	{"/", "search events (incremental, enter to keep, esc to clear)"},
	{"esc", "clear the active search"},
	{"j / k, ↓ / ↑", "move between events"},
	{"g / G", "first / last event"},
	{"f / b", "scroll the payload pane (pgdn / pgup also work)"},
	{"i / enter", "open details from the event list (finish search first)"},
	{"a", "toggle the full agent legend"},
	{"p", "filter projects (arrows move, enter selects)"},
	{"P", "all projects"},
	{"?", "close this help"},
}

// renderHelp draws the keybinding overlay, height-exact like every other pane.

// renderHelp draws the keybinding overlay, height-exact like every other pane.

// renderHelp draws the keybinding overlay, height-exact like every other pane.
func (m watchTUIModel) renderHelp(width, height int) string {
	// Derive the key column from the data so a new binding can never collide
	// with its own description.
	keyW := 0
	for _, row := range helpRows {
		if w := lipgloss.Width(row[0]); w > keyW {
			keyW = w
		}
	}
	// Leave room for the description; on a very narrow pane the key column has
	// to give way rather than push the row past the width.
	keyW = min(keyW, max(0, width-6))
	lines := []string{renderSectionHeader("Keys", width), ""}
	for _, row := range helpRows {
		lines = append(lines, renderHelpRow(row[0], row[1], width, keyW))
	}
	visible := max(1, height-2)
	window := lines
	if m.helpScroll > 0 {
		start := min(m.helpScroll, max(0, len(lines)-visible))
		window = lines[start:]
	}
	return fitRenderedLines(window, width, height)
}

func (m watchTUIModel) renderProjectPicker(width, height int) string {
	hint := "  type to filter · arrows move · enter select · esc cancel"
	lines := []string{
		renderSectionHeader("Projects", width),
		tuiMuted.Width(width).MaxWidth(width).Render(truncateCells(hint, width)),
	}
	matches := m.matchingProjects()
	if len(matches) == 0 {
		empty := "  no matching projects"
		lines = append(lines, tuiMuted.Width(width).MaxWidth(width).Render(truncateCells(empty, width)))
	}
	for i, project := range matches {
		style := tuiDefaultCell
		prefix := "  "
		if i == m.projectSelected {
			style = tuiSelectedRow
			prefix = "▸ "
		}
		// Project names are user supplied and unbounded: a long one must not
		// push the pane past the terminal width.
		cell := prefix + truncateCells(project, max(0, width-len([]rune(prefix))))
		lines = append(lines, style.Width(width).MaxWidth(width).Render(cell))
	}
	return fitRenderedLines(lines, width, height)
}

func renderHelpRow(keys, desc string, width, keyW int) string {
	if width <= 0 {
		return ""
	}
	keysStr := tuiLabel.Render(truncateCells(fmt.Sprintf("  %-*s", keyW+1, keys), width))
	// The key column may already fill the pane. Forcing at least one cell of
	// description there would make the row width+1, and lipgloss wraps a row
	// that is too long rather than clipping it, which pushes the whole view a
	// line past the terminal.
	valueW := max(0, width-lipgloss.Width(keysStr))
	if valueW == 0 {
		return keysStr
	}
	return keysStr + tuiDefaultCell.Width(valueW).MaxWidth(valueW).Render(truncateCells(desc, valueW))
}

// display returns the events the renderers should draw. With no active query
// this is the full buffer, so filtering costs a slice header, not a copy.

// eventMatches reports whether an event matches the search query. The query is
// split on spaces and every term must appear somewhere in the event.

// infoLabelW is the label column of the extended view.
const infoLabelW = 10

// renderEventInfo is the extended record of the selected event. The table row
// drops the date from the timestamp, shortens the agent to its tool identity
// and shows a one cell status icon, all so a long list stays scannable. This
// modal is where the untruncated values live, and it owns the whole terminal
// because there is nothing else to look at while it is open.
func (m watchTUIModel) renderEventInfo(width, height int) string {
	if height <= 0 || width <= 0 {
		return ""
	}
	display := m.display()
	if len(display) == 0 {
		return fitRenderedLines([]string{
			renderSectionHeader("Event", width),
			tuiMuted.Width(width).MaxWidth(width).Render(truncateCells("  no event selected", width)),
		}, width, height)
	}

	item := display[m.selected]
	// The label column and the value column are both derived from the pane, so
	// a narrow terminal shrinks the label instead of overflowing the value.
	labelW := min(infoLabelW, max(0, width-4))
	// valueW may legitimately be zero on a very narrow pane. Forcing one cell
	// here would make the row width+1, and lipgloss wraps rather than clips.
	valueW := max(0, width-labelW-4)
	indent := "    "
	if labelW+2 < width {
		indent = strings.Repeat(" ", labelW+2)
	}

	type row struct{ label, value string }
	rows := []row{
		{"Position", fmt.Sprintf("%d of %d in view", m.selected+1, len(display))},
		{"Time", item.Timestamp},
		{"", detailTS(item.Timestamp)},
		{"Agent", item.Agent},
		{"Host", item.Host},
		{"Project", item.Project},
		{"Task", item.Task},
		{"Type", item.Type},
		{"Status", statusIcon(item.Status) + " " + item.Status},
		{"File", item.File},
	}
	if qualifier := agentQualifier(item.Agent); qualifier != "" {
		rows = append(rows, row{"Session", normalizedAgent(item.Agent) + " · " + qualifier})
	}

	lines := []string{renderSectionHeader("Event", width), ""}
	emit := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		if label == "" {
			for _, l := range wrapWords(value, valueW) {
				lines = append(lines, tuiMuted.Width(width).MaxWidth(width).Render(indent+truncateCells(l, valueW)))
			}
			return
		}
		labelStr := tuiLabel.Render(truncateCells(fmt.Sprintf("  %-*s", labelW, label), width))
		if valueW == 0 {
			// No room beside the label: give the value its own full width rows.
			lines = append(lines, labelStr)
			for _, l := range wrapWords(value, max(1, width-2)) {
				lines = append(lines, tuiDefaultCell.Width(width).MaxWidth(width).Render("  "+truncateCells(l, max(0, width-2))))
			}
			return
		}
		wrapped := wrapWords(value, valueW)
		lines = append(lines, labelStr+tuiDefaultCell.Width(valueW).MaxWidth(valueW).Render(truncateCells(wrapped[0], valueW)))
		for _, l := range wrapped[1:] {
			lines = append(lines, indent+tuiDefaultCell.Width(valueW).MaxWidth(valueW).Render(truncateCells(l, valueW)))
		}
	}
	for _, r := range rows {
		emit(r.label, r.value)
	}
	if strings.TrimSpace(item.Detail) != "" {
		lines = append(lines, "")
		emit("Detail", item.Detail)
	}
	lines = append(lines, "")
	hint := "  i / enter / esc / q — back"
	lines = append(lines, tuiMuted.Width(width).MaxWidth(width).Render(truncateCells(hint, width)))

	return fitRenderedLines(lines, width, height)
}
