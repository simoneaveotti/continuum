package task

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"continuum/internal/events"
	"continuum/internal/setup"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Muted palette for agent colors — no neon
var agentPalette = []lipgloss.Color{
	lipgloss.Color("73"),  // steel teal
	lipgloss.Color("140"), // mauve
	lipgloss.Color("107"), // sage
	lipgloss.Color("179"), // amber
	lipgloss.Color("110"), // slate blue
	lipgloss.Color("174"), // dusty rose
	lipgloss.Color("115"), // seafoam
	lipgloss.Color("183"), // lavender
}

var (
	colorSep       = lipgloss.Color("237")
	colorMuted     = lipgloss.Color("241")
	colorDim       = lipgloss.Color("245")
	colorText      = lipgloss.Color("251")
	colorAccent    = lipgloss.Color("73")
	colorSelBg     = lipgloss.Color("238")
	colorBarBg     = lipgloss.Color("232")
	colorPayloadBg = lipgloss.Color("233")

	tuiBarStyle      = lipgloss.NewStyle().Foreground(colorDim).Background(colorBarBg).Padding(0, 1)
	tuiSectionTitle  = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	tuiSepStyle      = lipgloss.NewStyle().Foreground(colorSep)
	tuiHeaderRow     = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	tuiSelectedRow   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(colorSelBg).Bold(true)
	tuiMuted         = lipgloss.NewStyle().Foreground(colorMuted)
	tuiLabel         = lipgloss.NewStyle().Foreground(colorDim)
	tuiBorderColor   = colorSep // kept for compatibility
	tuiOkStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("71"))
	tuiErrStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("167"))
	tuiRunningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("136"))
	tuiDefaultStatus = lipgloss.NewStyle().Foreground(colorMuted)
	tuiDefaultCell   = lipgloss.NewStyle().Foreground(colorText)
	tuiPayloadLine   = lipgloss.NewStyle().Foreground(colorText).Background(colorPayloadBg)
)

type tuiTickMsg struct{}

const maxLegendAgents = 5

type watchTUIModel struct {
	projects    []string
	allProjects []string
	filterIndex int
	interval    time.Duration
	offset      int64
	events      []events.Event
	selected    int
	payloadTop  int
	width       int
	height      int
	showAllAgents bool
	agentColors map[string]lipgloss.Color
	agentOrder  []string
}

func WatchTUI(project string, interval time.Duration) error {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	allProjects, err := setup.ListProjects()
	if err != nil {
		return fmt.Errorf("cannot list projects: %w", err)
	}
	scope, err := watchProjects(project)
	if err != nil {
		return err
	}
	if project != "" && len(scope) == 0 {
		return fmt.Errorf("no projects found")
	}

	items, offset, err := events.ReadFromOffset(0)
	if err != nil {
		return err
	}

	model := watchTUIModel{
		projects:    scope,
		allProjects: allProjects,
		filterIndex: projectFilterIndex(scope, allProjects),
		interval:    interval,
		offset:      offset,
		events:      sortEventsNewestFirst(filterEvents(items, scope)),
		selected:    0,
		agentColors: map[string]lipgloss.Color{},
	}
	model.assignColors(model.events)
	model.trimEvents()

	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func (m watchTUIModel) Init() tea.Cmd {
	return tuiTickCmd(m.interval)
}

func (m watchTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				m.payloadTop = 0
			}
		case "down", "j":
			if m.selected < len(m.events)-1 {
				m.selected++
				m.payloadTop = 0
			}
		case "g":
			m.selected = 0
			m.payloadTop = 0
		case "G":
			if len(m.events) > 0 {
				m.selected = len(m.events) - 1
				m.payloadTop = 0
			}
		case "pgdown", "f":
			m.payloadTop += 10
			m.clampPayloadTop()
		case "pgup", "b":
			m.payloadTop -= 10
			if m.payloadTop < 0 {
				m.payloadTop = 0
			}
		case "p":
			m.cycleProjectFilter()
		case "P":
			m.setProjectFilter(nil)
		case "a":
			m.showAllAgents = !m.showAllAgents
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tuiTickMsg:
		wasAtTop := len(m.events) == 0 || m.selected == 0
		var selectedEvent events.Event
		hadSelection := len(m.events) > 0 && m.selected >= 0 && m.selected < len(m.events)
		if hadSelection {
			selectedEvent = m.events[m.selected]
		}

		items, offset, err := events.ReadFromOffset(m.offset)
		if err == nil && len(items) > 0 {
			filtered := filterEvents(items, m.projects)
			m.assignColors(filtered)
			m.events = sortEventsNewestFirst(append(m.events, filtered...))
			m.offset = offset
			m.trimEvents()
			if wasAtTop {
				m.selected = 0
				m.payloadTop = 0
			} else if hadSelection {
				next := findEventIndex(m.events, selectedEvent)
				if next != m.selected {
					m.payloadTop = 0
				}
				m.selected = next
			}
			if m.selected >= len(m.events) {
				m.selected = max(0, len(m.events)-1)
				m.payloadTop = 0
			}
		} else if err == nil {
			m.offset = offset
		}
		return m, tuiTickCmd(m.interval)
	}
	return m, nil
}

// View renders the layout:
//
//	header bar  (1 line)
//	agent legend (1 line)
//	[Events | Details]  (topH lines, split ~62/38)
//	─── separator ───  (1 line)
//	Payload  (bottomH lines, full width)
//	footer bar  (1 line)
func (m watchTUIModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading watch TUI..."
	}

	// chrome = header(1) + legend(1) + hSep(1) + footer(1) = 4
	bodyH := max(10, m.height-4)
	topH := (bodyH * 60) / 100
	bottomH := bodyH - topH

	leftW := (m.width * 70) / 100
	rightW := max(1, m.width-leftW-1) // -1 for vertical sep

	header := tuiBarStyle.Width(m.width).Render(m.renderHeaderBar())
	legend := lipgloss.NewStyle().Width(m.width).Render(m.renderLegend())

	eventsStr := m.renderEventsSection(leftW, topH)
	detailsStr := m.renderDetailsSection(rightW, topH)
	topSection := joinColumnsWithSep(eventsStr, detailsStr, topH)

	hSep := tuiSepStyle.Render(strings.Repeat("─", m.width))

	payloadStr := m.renderPayloadSection(m.width, bottomH)

	footer := tuiBarStyle.Width(m.width).Render(
		"  q quit  ·  j/k events  ·  g/G first/last  ·  f/b payload ↑↓  ·  a agents  ·  p next project  ·  P all",
	)

	return strings.Join([]string{header, legend, topSection, hSep, payloadStr, footer}, "\n")
}

func (m watchTUIModel) renderHeaderBar() string {
	pos := "0/0"
	if len(m.events) > 0 {
		pos = fmt.Sprintf("%d/%d", m.selected+1, len(m.events))
	}
	scopeLabel := "all projects"
	if len(m.projects) > 0 {
		scopeLabel = "project: " + strings.Join(m.projects, ",")
	}
	left := fmt.Sprintf("continuum watch  ·  %s  ·  %d events  ·  %d agents  ·  ↻ %s",
		scopeLabel, len(m.events), len(m.agentOrder), m.interval)
	// right-align pos; Padding(0,1) adds 2 chars so available = width-2
	available := m.width - 2
	gap := available - len(left) - len(pos)
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

func (m watchTUIModel) renderEventsSection(width, height int) string {
	titleLine := renderSectionHeader("Events", width)

	timeW := 16
	gapW := 1
	agentW := 10
	typeW := 12
	typeStatusGapW := 1
	statusW := 10
	targetW := max(12, width-timeW-gapW-agentW-typeW-typeStatusGapW-statusW-4)

	headerLine := lipgloss.JoinHorizontal(lipgloss.Left,
		renderCell("Time", timeW, tuiHeaderRow),
		renderCell("", gapW, tuiHeaderRow),
		renderCell("Agent", agentW, tuiHeaderRow),
		renderCell("Project/Task", targetW, tuiHeaderRow),
		renderCell("Type", typeW, tuiHeaderRow),
		renderCell("", typeStatusGapW, tuiHeaderRow),
		renderCell("Status", statusW, tuiHeaderRow),
	)
	sepLine := tuiSepStyle.Width(width).Render(strings.Repeat("─", width))

	fixed := []string{titleLine, headerLine, sepLine}
	rowsH := max(1, height-len(fixed))

	var eventRows []string
	if len(m.events) == 0 {
		eventRows = []string{tuiMuted.Width(width).Render("  no events yet")}
	} else {
		windowStart := m.selected - rowsH/2
		if windowStart < 0 {
			windowStart = 0
		}
		if windowStart+rowsH > len(m.events) {
			windowStart = max(0, len(m.events)-rowsH)
		}
		windowEnd := min(len(m.events), windowStart+rowsH)

		for i := windowStart; i < windowEnd; i++ {
			item := m.events[i]
			agent := normalizedAgent(item.Agent)
			target := eventTarget(item)
			baseStyle := tuiDefaultCell
			if i == m.selected {
				baseStyle = tuiSelectedRow
			}
			statusVal := statusIcon(item.Status) + " " + item.Status
			row := lipgloss.JoinHorizontal(lipgloss.Left,
				renderCell(shortTS(item.Timestamp), timeW, baseStyle),
				renderCell("", gapW, baseStyle),
				renderCell(agent, agentW, baseStyle.Copy().Foreground(m.agentColors[agent]).Bold(true)),
				renderCell(target, targetW, baseStyle),
				renderCell(item.Type, typeW, baseStyle),
				renderCell("", typeStatusGapW, baseStyle),
				renderCell(statusVal, statusW, mergeStyles(baseStyle, statusTextStyle(item.Status))),
			)
			eventRows = append(eventRows, row)
		}
	}

	lines := append(fixed, eventRows...)
	return fitRenderedLines(lines, width, height)
}

func (m watchTUIModel) renderDetailsSection(width, height int) string {
	titleLine := renderSectionHeader("Details", width)

	var detailLines []string
	if len(m.events) == 0 {
		detailLines = []string{tuiMuted.Render("  no event selected")}
	} else {
		item := m.events[m.selected]
		detailLines = []string{
			renderKV("Time", detailTS(item.Timestamp), width),
			renderKV("Agent", normalizedAgent(item.Agent), width),
			renderKV("Host", item.Host, width),
			renderKV("Project", item.Project, width),
			renderKV("Task", item.Task, width),
			renderKV("Type", item.Type, width),
			renderKV("Status", statusIcon(item.Status)+" "+item.Status, width),
		}
		if item.File != "" {
			detailLines = append(detailLines, renderKV("File", filepath.Base(item.File), width))
		}
	}

	lines := append([]string{titleLine}, detailLines...)
	return fitRenderedLines(lines, width, height)
}

func (m watchTUIModel) renderPayloadSection(width, height int) string {
	title := "Payload"
	if len(m.events) > 0 && m.events[m.selected].File != "" {
		title = "Payload  " + filepath.Base(m.events[m.selected].File)
	}

	var content string
	if len(m.events) == 0 {
		content = "no payload"
	} else {
		item := m.events[m.selected]
		content = strings.TrimSpace(readEventPayload(item))
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
	titleLine := renderSectionHeader(title+scrollIndicator, width)

	contentH := max(1, height-1)
	rawLines := fitWrappedLinesWindow(wrappedLines, contentH, m.payloadTop)

	// Render content lines with dark payload background
	rendered := make([]string, 0, height)
	rendered = append(rendered, titleLine)
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

func readEventPayload(item events.Event) string {
	if item.File == "" {
		return ""
	}
	fullPath := filepath.Join(setup.ContinuumPath(), filepath.FromSlash(item.File))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return ""
	}
	return string(data)
}

// renderSectionHeader renders "- Title -───────────────" spanning width.
func renderSectionHeader(title string, width int) string {
	titleStr := tuiSectionTitle.Render("- " + title + " -")
	titleW := lipgloss.Width(titleStr)
	lineW := max(0, width-titleW-1)
	return titleStr + tuiSepStyle.Render(strings.Repeat("─", lineW))
}

// joinColumnsWithSep merges two multi-line strings side by side with a │ separator.
func joinColumnsWithSep(left, right string, height int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	sep := tuiSepStyle.Render("│")
	rows := make([]string, height)
	for i := range rows {
		l, r := "", ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		rows[i] = l + sep + r
	}
	return strings.Join(rows, "\n")
}

// renderKV renders "  label:   value" with fixed label column.
func renderKV(label, value string, width int) string {
	labelStr := tuiLabel.Render(fmt.Sprintf("  %-9s", label+":"))
	labelW := lipgloss.Width(labelStr)
	valueW := max(1, width-labelW)
	return labelStr + tuiDefaultCell.Width(valueW).MaxWidth(valueW).Render(trim(value, valueW))
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

func (m *watchTUIModel) clampPayloadTop() {
	if len(m.events) == 0 || m.height == 0 {
		m.payloadTop = 0
		return
	}
	item := m.events[m.selected]
	content := strings.TrimSpace(readEventPayload(item))
	if content == "" {
		content = item.Detail
	}
	totalLines := len(wrapLines(content, max(1, m.width-2)))
	bodyH := max(10, m.height-4)
	topH := (bodyH * 60) / 100
	bottomH := bodyH - topH
	contentH := max(1, bottomH-1) // -1 for section title line
	maxTop := max(0, totalLines-contentH)
	if m.payloadTop > maxTop {
		m.payloadTop = maxTop
	}
}

func (m *watchTUIModel) trimEvents() {
	const maxEvents = 200
	if len(m.events) <= maxEvents {
		return
	}
	m.events = m.events[:maxEvents]
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
	m.events = sortEventsNewestFirst(filterEvents(readAllEvents(), m.projects))
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

func readAllEvents() []events.Event {
	items, _, err := events.ReadFromOffset(0)
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

func shortTS(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Format("2006-01-02 15:04")
}

func detailTS(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Format("2006-01-02 15:04:05")
}

func trim(value string, width int) string {
	if width <= 0 || len(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "…"
}

func fitLines(content string, width, height int) string {
	if height <= 0 || width <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	fitted := make([]string, 0, height)
	for _, line := range lines {
		fitted = append(fitted, trim(line, width))
		if len(fitted) == height {
			return strings.Join(fitted, "\n")
		}
	}
	for len(fitted) < height {
		fitted = append(fitted, "")
	}
	return strings.Join(fitted, "\n")
}

func fitLinesWindow(content string, width, height, start int) string {
	if height <= 0 || width <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if start < 0 {
		start = 0
	}
	if start > len(lines)-1 {
		start = max(0, len(lines)-1)
	}
	windowEnd := min(len(lines), start+height)
	fitted := make([]string, 0, height)
	for _, line := range lines[start:windowEnd] {
		fitted = append(fitted, trim(line, width))
	}
	for len(fitted) < height {
		fitted = append(fitted, "")
	}
	return strings.Join(fitted, "\n")
}

func wrapLines(content string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	lines := strings.Split(content, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			wrapped = append(wrapped, "")
			continue
		}
		for len(line) > width {
			wrapped = append(wrapped, line[:width])
			line = line[width:]
		}
		wrapped = append(wrapped, line)
	}
	if len(wrapped) == 0 {
		return []string{""}
	}
	return wrapped
}

func fitWrappedLinesWindow(lines []string, height, start int) []string {
	if height <= 0 {
		return nil
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	if start < 0 {
		start = 0
	}
	if start > len(lines)-1 {
		start = max(0, len(lines)-1)
	}
	windowEnd := min(len(lines), start+height)
	fitted := make([]string, 0, height)
	fitted = append(fitted, lines[start:windowEnd]...)
	for len(fitted) < height {
		fitted = append(fitted, "")
	}
	return fitted
}

func fitRenderedLines(lines []string, width, height int) string {
	if height <= 0 || width <= 0 {
		return ""
	}
	fitted := make([]string, 0, height)
	for _, line := range lines {
		fitted = append(fitted, lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line))
		if len(fitted) == height {
			return strings.Join(fitted, "\n")
		}
	}
	for len(fitted) < height {
		fitted = append(fitted, lipgloss.NewStyle().Width(width).Render(""))
	}
	return strings.Join(fitted, "\n")
}

func renderCell(value string, width int, style lipgloss.Style) string {
	return style.Width(width).MaxWidth(width).Render(trim(value, width))
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

func normalizedAgent(agent string) string {
	if strings.TrimSpace(agent) == "" {
		return "unknown"
	}
	return agent
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

func tuiTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg { return tuiTickMsg{} })
}
