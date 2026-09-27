// Payload access for the watch TUI: reading the stored body of the selected
// event and keeping the scroll position inside the visible content.
package task

import (
	"continuum/internal/events"
	"continuum/internal/filestore"
	"continuum/internal/setup"
	"os"
	"path/filepath"
	"strings"
)

// readEventPayload loads the stored body of an event.
//
// item.File comes from the activity log, so it is untrusted input: it is joined
// onto the storage path and must be proven to stay inside it before reading.
// Without this check a single log entry is enough to make the TUI display an
// arbitrary file on disk. This is the read-side counterpart of the project name
// validation added in ff3f215.
func readEventPayload(item events.Event) string {
	if item.File == "" {
		return ""
	}
	root := setup.ContinuumPath()
	fullPath := filepath.Join(root, filepath.FromSlash(item.File))
	if !filestore.WithinDir(root, fullPath) {
		return ""
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return ""
	}
	return string(data)
}

// payloadForSelection returns the cached payload of the selected event. It is
// resolved once per selection change in Update, never during render.

// payloadForSelection returns the cached payload of the selected event. It is
// resolved once per selection change in Update, never during render.

// payloadForSelection returns the cached payload of the selected event. It is
// resolved once per selection change in Update, never during render.
func (m watchTUIModel) payloadForSelection() string {
	if m.payloadValid && m.payloadFor == m.selected {
		return m.payload
	}
	display := m.display()
	if len(display) == 0 {
		return ""
	}
	return readEventPayload(display[m.selected])
}

// resolvePayload refreshes the payload cache for the current selection.

// resolvePayload refreshes the payload cache for the current selection.

// resolvePayload refreshes the payload cache for the current selection.
func (m *watchTUIModel) resolvePayload() {
	display := m.display()
	if len(display) == 0 {
		m.payload, m.payloadFor, m.payloadValid = "", 0, false
		return
	}
	m.payloadFor = m.selected
	m.payload = readEventPayload(display[m.selected])
	m.payloadValid = true
}

// trackSelection re-resolves the selected index after m.events changed. The
// event value alone is not a safe identity: Timestamp has second granularity,
// so two captures in the same second can be byte-identical and findEventIndex
// would jump the selection to the first of them.

// trackSelection re-resolves the selected index after m.events changed. The
// event value alone is not a safe identity: Timestamp has second granularity,
// so two captures in the same second can be byte-identical and findEventIndex
// would jump the selection to the first of them.

func (m *watchTUIModel) clampPayloadTop() {
	display := m.display()
	if len(display) == 0 || m.height == 0 {
		m.payloadTop = 0
		return
	}
	item := display[m.selected]
	content := strings.TrimSpace(m.payloadForSelection())
	if content == "" {
		content = item.Detail
	}
	totalLines := len(wrapLines(content, max(1, m.width-2)))
	_, _, bottomH := watchLayout(m.width, m.height)
	contentH := max(1, bottomH-1) // -1 for section title line
	maxTop := max(0, totalLines-contentH)
	if m.payloadTop > maxTop {
		m.payloadTop = maxTop
	}
}
