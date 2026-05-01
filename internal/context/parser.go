package context

import "continuum/internal/parse"

// ParseSections delegates to the shared parse package.
func ParseSections(content string) map[string]string {
	return parse.Sections(content)
}
