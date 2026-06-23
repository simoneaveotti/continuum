package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const indexFile = "index.md"
const indexHeader = "# Skills Index\n"

// IndexEntry holds a skill name and its one-line description.
type IndexEntry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ParseIndex parses the content of an index.md file into entries.
func ParseIndex(content string) []IndexEntry {
	var entries []IndexEntry
	lines := strings.Split(content, "\n")
	var current *IndexEntry
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if current != nil {
				entries = append(entries, *current)
			}
			name := strings.TrimPrefix(line, "## ")
			current = &IndexEntry{Name: strings.TrimSpace(name)}
			continue
		}
		if current != nil && current.Description == "" {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				current.Description = trimmed
			}
		}
	}
	if current != nil {
		entries = append(entries, *current)
	}
	return entries
}

// RenderIndex serialises entries into index.md content, sorted alphabetically.
func RenderIndex(entries []IndexEntry) string {
	sorted := make([]IndexEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	var sb strings.Builder
	sb.WriteString(indexHeader)
	for _, e := range sorted {
		sb.WriteString("\n## ")
		sb.WriteString(e.Name)
		sb.WriteString("\n")
		if e.Description != "" {
			sb.WriteString(e.Description)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// UpdateIndex adds or updates an entry for name in index.md.
// If description is empty and an entry already exists, keeps the existing description.
func UpdateIndex(basePath, name, description string) error {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return fmt.Errorf("cannot create skills directory: %w", err)
	}

	indexPath := filepath.Join(basePath, indexFile)
	var entries []IndexEntry
	if data, err := os.ReadFile(indexPath); err == nil {
		entries = ParseIndex(string(data))
	}

	found := false
	for i, e := range entries {
		if e.Name == name {
			if description != "" {
				entries[i].Description = description
			}
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, IndexEntry{Name: name, Description: description})
	}

	return os.WriteFile(indexPath, []byte(RenderIndex(entries)), 0o644)
}

// ListWithDescriptions returns skill entries from index.md when available,
// falling back to directory discovery with empty descriptions.
// The bool indicates whether the index was used.
func ListWithDescriptions(basePath string) ([]IndexEntry, bool, error) {
	indexPath := filepath.Join(basePath, indexFile)
	if data, err := os.ReadFile(indexPath); err == nil {
		return ParseIndex(string(data)), true, nil
	}

	names, err := List(basePath)
	if err != nil {
		return nil, false, err
	}
	entries := make([]IndexEntry, len(names))
	for i, n := range names {
		entries[i] = IndexEntry{Name: n}
	}
	return entries, false, nil
}

// MigrateAgentToIndex creates index.md if agent.md exists and index.md does not.
// Returns true if migration happened. Does not delete agent.md or copy its content.
func MigrateAgentToIndex(basePath string) (bool, error) {
	agentPath := filepath.Join(basePath, "agent.md")
	indexPath := filepath.Join(basePath, indexFile)

	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		return false, nil
	}
	if _, err := os.Stat(indexPath); err == nil {
		return false, nil
	}

	if err := os.WriteFile(indexPath, []byte(indexHeader+"\n"), 0o644); err != nil {
		return false, fmt.Errorf("cannot create index.md: %w", err)
	}
	return true, nil
}

// Delete removes a skill file (or directory) and its entry from index.md.
func Delete(basePath, name string) error {
	flatPath := filepath.Join(basePath, name+".md")
	dirPath := filepath.Join(basePath, name)
	skillDirFile := filepath.Join(dirPath, "SKILL.md")

	var deleteFn func() error
	switch {
	case fileExists(flatPath):
		deleteFn = func() error { return os.Remove(flatPath) }
	case fileExists(skillDirFile):
		deleteFn = func() error { return os.RemoveAll(dirPath) }
	default:
		return fmt.Errorf("skill %q not found. Run ctx skill list to see available skills", name)
	}

	if err := deleteFn(); err != nil {
		return fmt.Errorf("cannot delete skill %q: %w", name, err)
	}

	indexPath := filepath.Join(basePath, indexFile)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil // no index — skip silently
	}
	entries := ParseIndex(string(data))
	filtered := entries[:0]
	for _, e := range entries {
		if e.Name != name {
			filtered = append(filtered, e)
		}
	}
	return os.WriteFile(indexPath, []byte(RenderIndex(filtered)), 0o644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// List returns all skill names found in basePath.
// Discovers both flat (<name>.md) and directory (<name>/SKILL.md) layouts.
// Excludes the reserved index.md file.
func List(basePath string) ([]string, error) {
	entries, err := os.ReadDir(basePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			if strings.HasSuffix(e.Name(), ".md") {
				name := strings.TrimSuffix(e.Name(), ".md")
				if name == "index" {
					continue
				}
				seen[name] = true
			}
			continue
		}
		skillFile := filepath.Join(basePath, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			seen[e.Name()] = true
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// Show returns the content of a named skill.
// Tries <name>.md first, then <name>/SKILL.md.
// Special case: "agent" redirects to index if agent.md is absent.
func Show(basePath, name string) (string, error) {
	flatPath := filepath.Join(basePath, name+".md")
	if data, err := os.ReadFile(flatPath); err == nil {
		return string(data), nil
	}

	if name == "agent" {
		content, err := Show(basePath, "index")
		if err == nil {
			return "agent.md has been replaced by index.md\n\n" + content, nil
		}
	}

	dirPath := filepath.Join(basePath, name, "SKILL.md")
	if data, err := os.ReadFile(dirPath); err == nil {
		return string(data), nil
	}

	return "", fmt.Errorf("skill %q not found. Run ctx skill list to see available skills", name)
}

// Save writes content to <basePath>/<name>.md.
// Returns error if file exists and force is false.
// Creates index.md if agent.md exists and index.md does not (migration).
func Save(basePath, name, content string, force bool) error {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return fmt.Errorf("cannot create skills directory: %w", err)
	}

	dest := filepath.Join(basePath, name+".md")
	if _, err := os.Stat(dest); err == nil && !force {
		return fmt.Errorf("skill %q already exists. Use --yes to overwrite", name)
	}

	return os.WriteFile(dest, []byte(content), 0o644)
}
