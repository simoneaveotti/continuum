// Width- and height-aware text primitives. These know about cells and
// wrapping, not about events or layout policy.
package task

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// shortTS formats the timestamp for the event row. It shows clock time with
// seconds and drops the date: watch polls every couple of seconds, so
// minute-level precision cannot order events, and the saved date is always
// today. The full timestamp is still available via detailTS.
func shortTS(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Format("15:04:05")
}

func detailTS(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Format("2006-01-02 15:04:05")
}

// truncateCells shortens value to width display cells. Unlike trim it measures
// runes rather than bytes, so strings containing multi-byte glyphs (·, ↑↓, ✓)
// keep the number of visible columns they were laid out for.

// truncateCells shortens value to width display cells. Unlike trim it measures
// runes rather than bytes, so strings containing multi-byte glyphs (·, ↑↓, ✓)
// keep the number of visible columns they were laid out for.

// ansiEscape matches the SGR escape sequences lipgloss emits for styling.
var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

// truncateCells shortens value to width display cells. It measures runes
// rather than bytes, so strings containing multi-byte glyphs (·, ↑↓, ✓) keep
// the number of visible columns they were laid out for. Escape sequences are
// copied through without being counted, otherwise styled text would be
// truncated several columns early.
func truncateCells(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	var b strings.Builder
	used := 0
	for len(value) > 0 {
		if loc := ansiEscape.FindStringIndex(value); loc != nil && loc[0] == 0 {
			b.WriteString(value[loc[0]:loc[1]])
			value = value[loc[1]:]
			continue
		}
		r, size := utf8.DecodeRuneInString(value)
		w := lipgloss.Width(string(r))
		if used+w > width-1 {
			break
		}
		b.WriteRune(r)
		used += w
		value = value[size:]
	}
	return b.String() + "…"
}

func fitLines(content string, width, height int) string {
	if height <= 0 || width <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	fitted := make([]string, 0, height)
	for _, line := range lines {
		fitted = append(fitted, truncateCells(line, width))
		if len(fitted) == height {
			return strings.Join(fitted, "\n")
		}
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

// wrapWords breaks text on word boundaries instead of mid-word, which is what
// human readable values need. It preserves existing newlines and falls back to
// a hard wrap for a single word longer than the width, so a path or an
// identifier still gets broken rather than pushed out of the pane.
//
// This is wrapLines' counterpart: wrapLines is for pre-formatted payloads where
// a hard break at the exact column matters, wrapWords is for prose.
func wrapWords(content string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	var out []string
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}
		current := ""
		for _, word := range strings.Fields(line) {
			// A single word wider than the pane has to be hard broken.
			for lipgloss.Width(word) > width {
				if current != "" {
					out = append(out, current)
					current = ""
				}
				head := truncateCells(word, width+1)
				head = strings.TrimSuffix(head, "…")
				out = append(out, head)
				word = strings.TrimPrefix(word, head)
			}
			switch {
			case current == "":
				current = word
			case lipgloss.Width(current)+1+lipgloss.Width(word) <= width:
				current += " " + word
			default:
				out = append(out, current)
				current = word
			}
		}
		if current != "" {
			out = append(out, current)
		}
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}
