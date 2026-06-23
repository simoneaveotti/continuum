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
	VersionKey  = "CONTINUUM:BOOTSTRAP_VERSION"
)

type BootstrapCheck struct {
	File             string
	Status           string
	InstalledVersion string
	CurrentVersion   string
	Detail           string
}

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
			fmt.Fprintf(os.Stderr, "Skipping %s: %v\n", filename, err)
			continue
		}
		installed++
	}

	if installed == 0 {
		return fmt.Errorf("no agent instruction files found — create one of: AGENTS.md, CLAUDE.md, agent.md")
	}

	_ = events.Append(project, "", "agent_install", "ok", fmt.Sprintf("%d file(s)", installed))
	return nil
}

func Status(project string) ([]BootstrapCheck, error) {
	if err := setup.ValidateProjectName(project); err != nil {
		return nil, err
	}

	targetFiles, err := loadTargetFiles()
	if err != nil {
		return nil, err
	}

	checks := make([]BootstrapCheck, 0, len(targetFiles))
	for _, filename := range targetFiles {
		checks = append(checks, checkFile(filename))
	}
	return checks, nil
}

func Update(project string, force bool) (string, error) {
	if err := setup.ValidateProjectName(project); err != nil {
		return "", err
	}
	checks, err := Status(project)
	if err != nil {
		return "", err
	}
	if !force && !needsUpdate(checks) {
		return "Agent bootstrap already current.", nil
	}
	if err := setup.InitSession(true); err != nil {
		return "", err
	}
	if err := Install(project, true); err != nil {
		return "", err
	}
	return "Agent bootstrap updated.", nil
}

func needsUpdate(checks []BootstrapCheck) bool {
	for _, check := range checks {
		if check.Status == "stale" || check.Status == "unknown" {
			return true
		}
	}
	return false
}

func checkFile(filename string) BootstrapCheck {
	check := BootstrapCheck{
		File:           filename,
		CurrentVersion: template.BootstrapVersion,
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			check.Status = "missing"
			check.Detail = "file not found"
			return check
		}
		check.Status = "unknown"
		check.Detail = err.Error()
		return check
	}

	block, ok := extractBootstrapBlock(string(content))
	if !ok {
		check.Status = "missing"
		check.Detail = "no Continuum bootstrap block"
		return check
	}

	check.InstalledVersion = extractBootstrapVersion(block)
	if check.InstalledVersion == "" {
		check.Status = "unknown"
		check.Detail = "bootstrap version marker missing"
		return check
	}
	if check.InstalledVersion == template.BootstrapVersion {
		check.Status = "ok"
		return check
	}
	check.Status = "stale"
	return check
}

func extractBootstrapBlock(content string) (string, bool) {
	start := strings.Index(content, MarkerStart)
	end := strings.Index(content, MarkerEnd)
	if start == -1 || end == -1 || end <= start {
		return "", false
	}
	end += len(MarkerEnd)
	return content[start:end], true
}

func extractBootstrapVersion(block string) string {
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, VersionKey) {
			continue
		}
		line = strings.TrimPrefix(line, "<!--")
		line = strings.TrimSuffix(line, "-->")
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, VersionKey)
		return strings.TrimSpace(line)
	}
	return ""
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
		return fmt.Errorf("no Continuum bootstrap found to remove")
	}

	_ = events.Append("", "", "agent_remove", "ok", fmt.Sprintf("%d file(s)", removed))
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

	return nil
}
