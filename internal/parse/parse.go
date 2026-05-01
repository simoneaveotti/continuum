// Package parse provides shared markdown parsing utilities used across packages.
package parse

import (
	"strings"
)

// Sections parses a markdown document into a map of section title → content.
// Sections are delimited by ## headings; # headings reset the current section.
func Sections(content string) map[string]string {
	sections := make(map[string]string)

	var current string
	var builder strings.Builder

	flush := func() {
		if current != "" {
			sections[current] = strings.TrimSpace(builder.String())
			builder.Reset()
		}
	}

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			current = strings.TrimPrefix(line, "## ")
			continue
		}
		if strings.HasPrefix(line, "# ") {
			flush()
			current = ""
			continue
		}
		if current != "" {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}

	flush()
	return sections
}

// CleanValue strips markdown list prefixes, trims whitespace, and joins
// multiple lines with " | ". Returns "" for empty or placeholder-only values.
func CleanValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	lines := strings.Split(value, "\n")
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		}
		if line == "" || line == "-" {
			continue
		}
		cleaned = append(cleaned, line)
	}

	if len(cleaned) == 0 {
		return ""
	}
	return strings.Join(cleaned, " | ")
}
