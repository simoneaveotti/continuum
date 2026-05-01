package setup

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"continuum/internal/events"
	"continuum/internal/template"
)

func OnboardProject(project string, content []byte, force bool) error {
	if err := ValidateProjectName(project); err != nil {
		return err
	}

	base := ContinuumPath()
	if err := initBase(base, false); err != nil {
		return err
	}
	if err := ensureGitRepo(base, false); err != nil {
		return err
	}

	projectDir := filepath.Join(base, "projects", project)
	projectPath := filepath.Join(projectDir, "project.md")
	projectTasksDir := filepath.Join(projectDir, "tasks")
	if err := ensureDir(projectTasksDir); err != nil {
		return fmt.Errorf("cannot create project directory: %w", err)
	}

	normalized := normalizeProjectContent(content)
	if len(normalized) == 0 {
		return fmt.Errorf("project onboarding content is empty")
	}

	existing, err := os.ReadFile(projectPath)
	switch {
	case err == nil:
		if !force && hasRealProjectContent(project, existing) {
			return fmt.Errorf("project context already exists; rerun with --force to replace it")
		}
	case os.IsNotExist(err):
		// nothing to preserve
	default:
		return fmt.Errorf("cannot read existing project context: %w", err)
	}

	if err := os.WriteFile(projectPath, normalized, 0o644); err != nil {
		return fmt.Errorf("cannot write project context: %w", err)
	}

	files := []string{filepath.ToSlash(filepath.Join("projects", project, "project.md"))}
	if err := events.Append(project, "", "project_onboarded", "ok", "project context updated"); err == nil {
		files = append([]string{events.ActivityRelPath()}, files...)
	}
	if err := CommitFiles(fmt.Sprintf("onboard(%s): project context updated", project), files); err != nil {
		return err
	}
	PushBestEffort()
	return nil
}

func normalizeProjectContent(content []byte) []byte {
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return nil
	}
	return append([]byte(trimmed), '\n')
}

func hasRealProjectContent(project string, content []byte) bool {
	templateData, err := template.GetProject(project)
	if err != nil {
		return len(bytes.TrimSpace(content)) > 0
	}
	return !bytes.Equal(normalizeProjectContent(content), normalizeProjectContent(templateData))
}
