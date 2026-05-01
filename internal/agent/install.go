package agent

import (
	"fmt"
	"os"
	"strings"

	"continuum/internal/events"
	"continuum/internal/setup"
	"continuum/internal/template"
)

const (
	MarkerStart = "<!-- CONTINUUM:START -->"
	MarkerEnd   = "<!-- CONTINUUM:END -->"
)

func loadTargetFiles() ([]string, error) {
	path := setup.ResolvePath("agent-targets.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("agent-targets.txt not found — run 'ctx init' first")
		}
		return nil, fmt.Errorf("cannot read agent-targets.txt: %w", err)
	}

	var targets []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		targets = append(targets, line)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("agent-targets.txt is empty — add at least one target filename")
	}

	return targets, nil
}

func Install(project string, force bool) error {
	if err := setup.ValidateProjectName(project); err != nil {
		return err
	}

	targetFiles, err := loadTargetFiles()
	if err != nil {
		return err
	}

	installed := 0

	for _, filename := range targetFiles {
		if err := installToFile(filename, project, force); err != nil {
			fmt.Printf("Skipping %s: %v\n", filename, err)
			continue
		}
		installed++
	}

	if installed == 0 {
		fmt.Println("No agent instruction files found.")
		fmt.Println("Create one of: AGENTS.md, CLAUDE.md, agent.md")
		return nil
	}

	_ = events.Append(project, "", "agent_install", "ok", fmt.Sprintf("%d file(s)", installed))
	fmt.Printf("Installed Continuum bootstrap (project: %s) to %d file(s).\n", project, installed)
	return nil
}

func installToFile(filename, project string, force bool) error {
	path := filename

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found")
	}
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read: %w", err)
	}

	existing := string(content)

	if strings.Contains(existing, MarkerStart) && strings.Contains(existing, MarkerEnd) {
		if !force {
			fmt.Printf("%s: already has Continuum bootstrap (skipping)\n", filename)
			return nil
		}
		start := strings.Index(existing, MarkerStart)
		end := strings.Index(existing, MarkerEnd)
		if end > start {
			existing = strings.TrimRight(existing[:start]+existing[end+len(MarkerEnd):], " \n\t")
		}
	}

	bootstrap, err := template.GetBootstrap(project)
	if err != nil {
		return fmt.Errorf("cannot get bootstrap template: %w", err)
	}

	newContent := strings.TrimRight(existing, " \n\t") + "\n\n" + string(bootstrap)

	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("cannot write: %w", err)
	}

	fmt.Printf("%s: added Continuum bootstrap\n", filename)
	return nil
}

func Remove() error {
	targetFiles, err := loadTargetFiles()
	if err != nil {
		return err
	}

	removed := 0

	for _, filename := range targetFiles {
		if err := removeFromFile(filename); err != nil {
			continue
		}
		removed++
	}

	if removed == 0 {
		fmt.Println("No Continuum bootstrap found to remove.")
		return nil
	}

	_ = events.Append("", "", "agent_remove", "ok", fmt.Sprintf("%d file(s)", removed))
	fmt.Printf("Removed Continuum bootstrap from %d file(s).\n", removed)
	return nil
}

func removeFromFile(filename string) error {
	path := filename

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	existing := string(content)

	start := strings.Index(existing, MarkerStart)
	end := strings.Index(existing, MarkerEnd)

	if start == -1 || end == -1 {
		return fmt.Errorf("no bootstrap found")
	}

	end += len(MarkerEnd)
	newContent := existing[:start] + existing[end:]

	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return err
	}

	fmt.Printf("%s: removed Continuum bootstrap\n", filename)
	return nil
}
