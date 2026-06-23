// Package parse provides shared markdown parsing utilities used across packages.
package parse

import (
	"strings"
)

// sectionAliases maps common field names to alternative section matching terms.
// This allows querying by a canonical name while matching sections with slightly
// different headings (e.g. "locked decisions" matches "## Decisions (Locked)").
var sectionAliases = map[string]string{
	"what was done":         "what",
	"next recommended step": "next",
	"locked decisions":      "decision",
	"open issues":           "active",
}

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

// matchSection reports whether a heading line matches the given field name.
// Matching is case-insensitive; field-specific aliases are also checked.
func matchSection(heading, field string) bool {
	lower := strings.ToLower(strings.TrimSpace(heading))
	fieldLower := strings.ToLower(strings.TrimSpace(field))

	if strings.Contains(lower, fieldLower) {
		return true
	}
	if alias, ok := sectionAliases[fieldLower]; ok {
		return strings.Contains(lower, alias)
	}
	return false
}

// ExtractField finds the first non-empty content line from the section whose
// heading matches fieldName. Section matching is case-insensitive with aliases.
func ExtractField(content, fieldName string) string {
	inSection := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		if strings.HasPrefix(lower, "## ") || strings.HasPrefix(lower, "# ") {
			if inSection {
				break
			}
			if matchSection(trimmed, fieldName) {
				inSection = true
			}
			continue
		}

		if inSection {
			cleaned := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "-"), "*"))
			if cleaned != "" && !strings.HasPrefix(cleaned, "#") {
				return cleaned
			}
		}
	}
	return ""
}

// ExtractBulletList extracts bullet list items from the section matching
// sectionName. Each line prefixed with - or * is returned as a cleaned string.
func ExtractBulletList(content, sectionName string) []string {
	var lines []string
	inSection := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		if strings.HasPrefix(lower, "## ") || strings.HasPrefix(lower, "# ") {
			if inSection {
				break
			}
			if matchSection(trimmed, sectionName) {
				inSection = true
			}
			continue
		}

		if inSection {
			cleaned := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "-"), "*"))
			if cleaned != "" && !strings.HasPrefix(cleaned, "#") {
				lines = append(lines, cleaned)
			}
		}
	}

	return lines
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
