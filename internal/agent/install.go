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

// FileResult reports what happened to a single target file during Install/Update.
// Status is one of "installed", "skipped", or "error".
type FileResult struct {
	Filename string
	Status   string
	Detail   string
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

func Install(project string, force bool) ([]FileResult, error) {
	if err := setup.ValidateProjectName(project); err != nil {
		return nil, err
	}

	targetFiles, err := loadTargetFiles()
	if err != nil {
		return nil, err
	}

	results := make([]FileResult, 0, len(targetFiles))
	installedCount := 0
	processed := 0

	for _, filename := range targetFiles {
		skipped, err := installToFile(filename, project, force)
		switch {
		case err != nil && err.Error() == "file not found":
			results = append(results, FileResult{Filename: filename, Status: "skipped", Detail: "not found"})
		case err != nil:
			results = append(results, FileResult{Filename: filename, Status: "error", Detail: err.Error()})
		case skipped:
			results = append(results, FileResult{Filename: filename, Status: "skipped", Detail: "already current"})
			processed++
		default:
			results = append(results, FileResult{Filename: filename, Status: "installed"})
			installedCount++
			processed++
		}
	}

	if processed == 0 {
		return results, fmt.Errorf("no agent instruction files found — create one of: AGENTS.md, CLAUDE.md, agent.md")
	}

	_ = events.Append(project, "", "agent_install", "ok", fmt.Sprintf("%d file(s)", installedCount))
	return results, nil
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

// Update refreshes stale bootstrap instructions. It returns nil results when
// nothing needed updating, otherwise the per-file outcome of the re-install.
func Update(project string, force bool) ([]FileResult, error) {
	if err := setup.ValidateProjectName(project); err != nil {
		return nil, err
	}
	checks, err := Status(project)
	if err != nil {
		return nil, err
	}
	if !force && !needsUpdate(checks) {
		return nil, nil
	}
	if err := setup.InitSession(true); err != nil {
		return nil, err
	}
	return Install(project, true)
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

// installToFile writes the bootstrap block into filename. It returns
// (true, nil) when the file already has a current block and force is false,
// meaning nothing was written.
func installToFile(filename, project string, force bool) (bool, error) {
	path := filename

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, fmt.Errorf("file not found")
	}
	if err != nil {
		return false, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("cannot read: %w", err)
	}

	existing := string(content)

	if strings.Contains(existing, MarkerStart) && strings.Contains(existing, MarkerEnd) {
		if !force {
			return true, nil
		}
		start := strings.Index(existing, MarkerStart)
		end := strings.Index(existing, MarkerEnd)
		if end > start {
			existing = strings.TrimRight(existing[:start]+existing[end+len(MarkerEnd):], " \n\t")
		}
	}

	bootstrap, err := template.GetBootstrap(project)
	if err != nil {
		return false, fmt.Errorf("cannot get bootstrap template: %w", err)
	}

	newContent := strings.TrimRight(existing, " \n\t") + "\n\n" + string(bootstrap)

	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return false, fmt.Errorf("cannot write: %w", err)
	}

	return false, nil
}

func Remove() ([]FileResult, error) {
	targetFiles, err := loadTargetFiles()
	if err != nil {
		return nil, err
	}

	results := make([]FileResult, 0, len(targetFiles))
	removed := 0

	for _, filename := range targetFiles {
		err := removeFromFile(filename)
		switch {
		case os.IsNotExist(err):
			results = append(results, FileResult{Filename: filename, Status: "skipped", Detail: "not found"})
		case err != nil && err.Error() == "no bootstrap found":
			results = append(results, FileResult{Filename: filename, Status: "skipped", Detail: "no bootstrap found"})
		case err != nil:
			results = append(results, FileResult{Filename: filename, Status: "error", Detail: err.Error()})
		default:
			results = append(results, FileResult{Filename: filename, Status: "removed"})
			removed++
		}
	}

	if removed == 0 {
		return results, fmt.Errorf("no Continuum bootstrap found to remove")
	}

	_ = events.Append("", "", "agent_remove", "ok", fmt.Sprintf("%d file(s)", removed))
	return results, nil
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
