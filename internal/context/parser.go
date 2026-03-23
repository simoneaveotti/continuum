package context

import (
	"strings"
)

func ParseSections(content string) map[string]string {
	sections := make(map[string]string)

	lines := strings.Split(content, "\n")

	var current string
	var builder strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if current != "" {
				sections[current] = strings.TrimSpace(builder.String())
				builder.Reset()
			}
			current = strings.TrimPrefix(line, "## ")
			continue
		}

		if current != "" {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}

	if current != "" {
		sections[current] = strings.TrimSpace(builder.String())
	}

	return sections
}
