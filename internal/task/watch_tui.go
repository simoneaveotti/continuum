package task

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"continuum/internal/events"
	"continuum/internal/setup"
)

type tuiTickMsg struct{}

const maxLegendAgents = 5

type watchTUIModel struct {
	projects      []string
	allProjects   []string
	filterIndex   int
	interval      time.Duration
	offset        int64
	events        []events.Event
	selected      int
	payloadTop    int
	width         int
	height        int
	showAllAgents bool
	agentColors   map[string]lipgloss.Color
	agentOrder    []string

	// payload caches the resolved body of the selected event. View() must never
	// read it from disk: View runs on every tick, keypress and resize, and a
	// payload can be a full context dump.
	payload      string
	payloadFor   int
	payloadValid bool

	// watchErr holds the last activity-log read failure so the TUI can show it
	// instead of silently retrying.
	watchErr string
	// dropped counts events discarded by trimEvents, surfaced in the header.
	dropped int

	// search state. matched mirrors events, filtered by query; the renderers
	// read through display() so an empty query costs nothing.
	searching bool
	query     string
	draft     string
	matched   []events.Event

	projectPicking  bool
	projectDraft    string
	projectSelected int

	// help toggles the keybinding overlay.
	help       bool
	helpScroll int

	// info shows the extended record of the selected event. It is a modal: the
	// table row elides precision on purpose, and this is where the full values
	// are available without giving them a permanent column.
	info bool
}

// recomputeDraft applies the in-progress search text without leaving search
// mode, so the list narrows live as the user types.

// display returns the events the renderers should draw. With no active query
// this is the full buffer, so filtering costs a slice header, not a copy.

// display returns the events the renderers should draw. With no active query
// this is the full buffer, so filtering costs a slice header, not a copy.
// watchLayout is the single source of truth for the vertical split. View and
// clampPayloadTop must agree, otherwise the payload can be scrolled past the
// last line that is actually on screen.

// watchLayout is the single source of truth for the vertical split. View and
// clampPayloadTop must agree, otherwise the payload can be scrolled past the
// last line that is actually on screen.

// watchLayout is the single source of truth for the vertical split. View and
// clampPayloadTop must agree, otherwise the payload can be scrolled past the
// last line that is actually on screen.
func watchLayout(width, height int) (bodyH, topH, bottomH int) {
	_ = width
	// chrome = header(1) + legend(1) + hSep(1) + footer(1). The body must be
	// exactly height-4: a floor here makes the view taller than the terminal
	// on short screens, which is the scroll drift this layout exists to avoid.
	bodyH = max(0, height-4)
	if bodyH < 2 {
		return bodyH, 0, 0
	}
	topH = max(1, (bodyH*60)/100)
	bottomH = bodyH - topH
	if bottomH < 1 {
		bottomH = 1
		topH = bodyH - 1
	}
	return bodyH, topH, bottomH
}

// watchColumns is the single source of truth for the horizontal split.

// watchColumns is the single source of truth for the horizontal split.

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

	// Read only the tail: the activity log grows without bound and parsing all
	// of it to keep the newest maxWatchEvents wasted ~40x the work.
	items, offset, err := events.ReadTail(maxWatchEvents)
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
		if m.projectPicking {
			return m.updateProjectPicker(msg)
		}
		if m.searching {
			return m.updateSearch(msg)
		}
		if m.info {
			switch msg.String() {
			case "q", "ctrl+c", "esc", "i", "enter":
				m.info = false
			}
			return m, nil
		}
		if m.help {
			switch msg.String() {
			case "q", "ctrl+c", "esc", "?":
				m.help = false
			case "j", "down":
				m.helpScroll++
			case "k", "up":
				m.helpScroll = max(0, m.helpScroll-1)
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.help = true
			m.helpScroll = 0
			return m, nil
		case "i", "enter":
			if len(m.display()) > 0 {
				m.info = true
			}
			return m, nil
		case "/":
			m.searching = true
			m.draft = m.query
			m.recomputeDraft()
			return m, nil
		case "esc":
			if m.query != "" {
				m.query = ""
				m.draft = ""
				m.matched = nil
				m.clampSelection()
				m.resolvePayload()
			}
			return m, nil
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				m.payloadTop = 0
				m.resolvePayload()
			}
		case "down", "j":
			if m.selected < len(m.display())-1 {
				m.selected++
				m.payloadTop = 0
				m.resolvePayload()
			}
		case "g":
			m.selected = 0
			m.payloadTop = 0
			m.resolvePayload()
		case "G":
			if display := m.display(); len(display) > 0 {
				m.selected = len(display) - 1
				m.payloadTop = 0
				m.resolvePayload()
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
			m.projectPicking = true
			m.projectDraft = ""
			m.projectSelected = 0
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
		previous := m.selected
		hadSelection := len(m.events) > 0 && m.selected >= 0 && m.selected < len(m.events)

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
				if next := m.trackSelection(previous); next != previous {
					m.payloadTop = 0
				}
			}
			if m.selected >= len(m.events) {
				m.selected = max(0, len(m.events)-1)
				m.payloadTop = 0
			}
			m.resolvePayload()
		} else if err == nil {
			m.offset = offset
		} else if errors.Is(err, events.ErrLogRewritten) {
			// The log changed under us, usually ctx repair --activity. Re-read
			// the tail and say so: a silent resync is indistinguishable from an
			// agent that stopped working.
			m.resyncAfterRewrite()
		} else {
			// Surface read failures instead of silently retrying forever.
			m.watchErr = err.Error()
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

// View renders the layout:
//
//	header bar  (1 line)
//	agent legend (1 line)
//	[Events | Details]  (topH lines, split ~62/38)
//	─── separator ───  (1 line)
//	Payload  (bottomH lines, full width)
//	footer bar  (1 line)

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
	if m.info {
		return m.renderEventInfo(m.width, m.height)
	}

	_, topH, bottomH := watchLayout(m.width, m.height)
	header := tuiBarStyle.Width(m.width).MaxWidth(m.width).Render(m.renderHeaderBar())
	// Like the footer, the legend must be truncated rather than wrapped: an extra
	// line anywhere makes the whole view one row taller than the terminal.
	legend := lipgloss.NewStyle().Width(m.width).MaxWidth(m.width).Render(
		truncateCells(m.renderLegend(), m.width),
	)

	var topSection string
	if m.projectPicking {
		topSection = m.renderProjectPicker(m.width, topH)
	} else if m.help {
		topSection = m.renderHelp(m.width, topH)
	} else {
		// The event table spans the full width: Detail lives in a row and
		// Status is a single icon cell, so there is nothing left to justify a
		// second column.
		topSection = m.renderEventsSection(m.width, topH)
	}

	hSep := tuiSepStyle.Width(m.width).MaxWidth(m.width).Render(strings.Repeat("─", m.width))

	payloadStr := m.renderPayloadSection(m.width, bottomH)

	// The footer must never wrap: an extra line here makes the whole view one
	// line taller than the terminal, which scrolls the alt-screen on every
	// frame and slowly walks the layout off screen.
	footerText := "  q quit  ·  / search  ·  j/k events  ·  f/b payload  ·  g/G ends  ·  i details  ·  a agents  ·  ? help"
	if m.searching {
		footerText = "  /" + m.draft + "▏   (enter keep · esc cancel)"
	} else if m.projectPicking {
		footerText = "  p " + m.projectDraft + "▏   (enter select · esc cancel)"
	}
	footer := tuiBarStyle.Width(m.width).MaxWidth(m.width).Render(
		truncateCells(footerText, m.width-2),
	)

	return strings.Join([]string{header, legend, topSection, hSep, payloadStr, footer}, "\n")
}

func (m watchTUIModel) updateProjectPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.projectPicking = false
	case tea.KeyEnter:
		matches := m.matchingProjects()
		if len(matches) > 0 {
			m.setProjectFilter([]string{matches[m.clampProjectSelection(m.projectSelected)]})
		}
		m.projectPicking = false
	case tea.KeyUp:
		m.projectSelected = max(0, m.projectSelected-1)
	case tea.KeyDown:
		m.projectSelected = m.clampProjectSelection(m.projectSelected + 1)
	case tea.KeyBackspace:
		if runes := []rune(m.projectDraft); len(runes) > 0 {
			m.projectDraft = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		m.projectDraft += string(msg.Runes)
		m.projectSelected = 0
	case tea.KeyCtrlC:
		return m, tea.Quit
	}
	return m, nil
}

// clampProjectSelection keeps the highlighted project inside the match list.
// It never returns a negative: a negative index stored on the model is what let
// Enter reach matches[-1] and panic. With nothing matching the index is
// meaningless, so it pins to 0 and callers must check the list is not empty.
func (m watchTUIModel) clampProjectSelection(candidate int) int {
	limit := len(m.matchingProjects())
	if limit == 0 {
		return 0
	}
	return max(0, min(limit-1, candidate))
}

func (m watchTUIModel) matchingProjects() []string {
	needle := strings.ToLower(m.projectDraft)
	var matches []string
	for _, project := range m.allProjects {
		if strings.Contains(strings.ToLower(project), needle) {
			matches = append(matches, project)
		}
	}
	return matches
}

func tuiTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg { return tuiTickMsg{} })
}

// recomputeDraft applies the in-progress search text without leaving search
// mode, so the list narrows live as the user types.
func (m *watchTUIModel) recomputeDraft() {
	m.query = m.draft
	m.recomputeMatch()
	m.clampSelection()
	m.resolvePayload()
}

// updateSearch handles keys while the search prompt is open.

// updateSearch handles keys while the search prompt is open.

// updateSearch handles keys while the search prompt is open.
func (m watchTUIModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Abandon the edit, keep the previous committed filter.
		m.searching = false
		m.draft = m.query
		return m, nil
	case tea.KeyEnter:
		m.searching = false
		m.query = m.draft
		m.recomputeMatch()
		m.clampSelection()
		m.resolvePayload()
		return m, nil
	case tea.KeyBackspace:
		if m.draft != "" {
			runes := []rune(m.draft)
			m.draft = string(runes[:len(runes)-1])
		}
	case tea.KeySpace:
		m.draft += " "
	case tea.KeyRunes:
		m.draft += string(msg.Runes)
	case tea.KeyCtrlC:
		return m, tea.Quit
	default:
		return m, nil
	}
	m.recomputeDraft()
	return m, nil
}

// helpRows is the content of the ? overlay.

// helpRows is the content of the ? overlay.
