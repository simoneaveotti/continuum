package task

import (
	"bufio"
	"fmt"
	"strings"
)

func promptWithDefault(reader *bufio.Reader, label, current string) (string, error) {
	if current != "" {
		fmt.Printf("%s [%s]: ", label, current)
	} else {
		fmt.Printf("%s: ", label)
	}

	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	text = strings.TrimSpace(text)

	if text == "" {
		return current, nil
	}

	return text, nil
}

func parseSections(content string) map[string]string {
	sections := make(map[string]string)

	lines := strings.Split(content, "\n")

	var current string
	var builder strings.Builder

	flush := func() {
		if current != "" {
			sections[current] = strings.TrimSpace(builder.String())
			builder.Reset()
		}
	}

	for _, line := range lines {
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

func cleanPrefill(value string) string {
	value = strings.TrimSpace(value)

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

	return strings.Join(cleaned, " | ")
}
