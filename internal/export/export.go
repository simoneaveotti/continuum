// Package export provides zip-based export and import of Continuum task,
// project, and session data, with optional AES-GCM encryption.
package export

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"continuum/internal/events"
	"continuum/internal/setup"
)

type EncryptionAlgo string
type ArchiveKind string

const (
	AlgoAES_GCM_V2 EncryptionAlgo = "aes-gcm-v2"

	ArchiveTask    ArchiveKind = "task"
	ArchiveProject ArchiveKind = "project"
	ArchiveSession ArchiveKind = "session"
)

const (
	v2Magic        = "CTXENC2"
	v2SaltSize     = 16
	v2KeySize      = 32
	v2ArgonTime    = uint32(3)
	v2ArgonMemory  = uint32(32 * 1024)
	v2ArgonThreads = uint8(1)
	manifestName   = "CONTINUUM_EXPORT.json"
)

type ArchiveManifest struct {
	Kind     ArchiveKind `json:"kind"`
	Task     string      `json:"task,omitempty"`
	Projects []string    `json:"projects,omitempty"`
}

func (e EncryptionAlgo) String() string {
	return string(e)
}

func (e EncryptionAlgo) Valid() bool {
	return e == "" || e == AlgoAES_GCM_V2
}

func (e EncryptionAlgo) Default() EncryptionAlgo {
	if e == "" {
		return AlgoAES_GCM_V2
	}
	return e
}

func validateProjects(projects []string) ([]string, error) {
	if len(projects) == 0 {
		return nil, fmt.Errorf("at least one project is required")
	}
	seen := map[string]bool{}
	var out []string
	for _, project := range projects {
		if err := setup.ValidateProjectName(project); err != nil {
			return nil, err
		}
		if seen[project] {
			continue
		}
		seen[project] = true
		projectDir := filepath.Join(setup.ContinuumPath(), "projects", project)
		if _, err := os.Stat(projectDir); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("project not found: %s", project)
			}
			return nil, fmt.Errorf("cannot access project directory: %w", err)
		}
		out = append(out, project)
	}
	sort.Strings(out)
	return out, nil
}

func taskArchivePaths(task, project string) ([]string, error) {
	if err := setup.ValidateTaskName(task); err != nil {
		return nil, err
	}
	projects, err := validateProjects([]string{project})
	if err != nil {
		return nil, err
	}
	project = projects[0]
	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	if _, err := os.Stat(taskDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task not found: %s", task)
		}
		return nil, fmt.Errorf("cannot access task directory: %w", err)
	}

	paths := []string{
		filepath.ToSlash(filepath.Join("projects", project, "project.md")),
		filepath.ToSlash(filepath.Join("projects", project, "tasks", task)),
	}
	return paths, nil
}

func sessionArchivePaths() ([]string, error) {
	root := setup.ContinuumPath()
	candidates := []string{"profile.md", "agent-targets.txt", "skills", "templates", "projects", "events"}
	var paths []string
	for _, rel := range candidates {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			paths = append(paths, rel)
		}
	}
	return paths, nil
}

func ExportTask(task, project, outputPath string) (string, error) {
	paths, err := taskArchivePaths(task, project)
	if err != nil {
		return "", err
	}
	output, err := writeZipArchive(paths, outputPath, task, ArchiveManifest{
		Kind:     ArchiveTask,
		Task:     task,
		Projects: []string{project},
	})
	if err != nil {
		return "", err
	}
	_ = events.Append(project, task, "export", "ok", "task archive written")
	return output, nil
}

func ExportTaskEncrypted(task, project, outputPath string, algo EncryptionAlgo) (string, error) {
	paths, err := taskArchivePaths(task, project)
	if err != nil {
		return "", err
	}
	output, err := writeEncryptedZipArchive(paths, outputPath, task, algo, ArchiveManifest{
		Kind:     ArchiveTask,
		Task:     task,
		Projects: []string{project},
	})
	if err != nil {
		return "", err
	}
	_ = events.Append(project, task, "export", "ok", "encrypted task archive written")
	return output, nil
}

func ExportProjects(projects []string, outputPath string) (string, error) {
	projects, err := validateProjects(projects)
	if err != nil {
		return "", err
	}
	var paths []string
	for _, project := range projects {
		paths = append(paths, filepath.ToSlash(filepath.Join("projects", project)))
	}
	base := "projects"
	if len(projects) == 1 {
		base = "project-" + projects[0]
	}
	output, err := writeZipArchive(paths, outputPath, base, ArchiveManifest{
		Kind:     ArchiveProject,
		Projects: projects,
	})
	if err != nil {
		return "", err
	}
	_ = events.Append(exportProjectForEvent(projects), "", "export", "ok", exportProjectDetail(projects, false))
	return output, nil
}

func ExportProjectsEncrypted(projects []string, outputPath string, algo EncryptionAlgo) (string, error) {
	projects, err := validateProjects(projects)
	if err != nil {
		return "", err
	}
	var paths []string
	for _, project := range projects {
		paths = append(paths, filepath.ToSlash(filepath.Join("projects", project)))
	}
	base := "projects"
	if len(projects) == 1 {
		base = "project-" + projects[0]
	}
	output, err := writeEncryptedZipArchive(paths, outputPath, base, algo, ArchiveManifest{
		Kind:     ArchiveProject,
		Projects: projects,
	})
	if err != nil {
		return "", err
	}
	_ = events.Append(exportProjectForEvent(projects), "", "export", "ok", exportProjectDetail(projects, true))
	return output, nil
}

func ExportSession(outputPath string) (string, error) {
	paths, err := sessionArchivePaths()
	if err != nil {
		return "", err
	}
	output, err := writeZipArchive(paths, outputPath, "session", ArchiveManifest{Kind: ArchiveSession})
	if err != nil {
		return "", err
	}
	_ = events.Append("", "", "export", "ok", "session archive written")
	return output, nil
}

func ExportSessionEncrypted(outputPath string, algo EncryptionAlgo) (string, error) {
	paths, err := sessionArchivePaths()
	if err != nil {
		return "", err
	}
	output, err := writeEncryptedZipArchive(paths, outputPath, "session", algo, ArchiveManifest{Kind: ArchiveSession})
	if err != nil {
		return "", err
	}
	_ = events.Append("", "", "export", "ok", "encrypted session archive written")
	return output, nil
}

func exportProjectForEvent(projects []string) string {
	if len(projects) == 1 {
		return projects[0]
	}
	return ""
}

func manifestProjectForEvent(manifest *ArchiveManifest) string {
	if manifest == nil || len(manifest.Projects) != 1 {
		return ""
	}
	return manifest.Projects[0]
}

func exportProjectDetail(projects []string, encrypted bool) string {
	scope := fmt.Sprintf("%d projects archive written", len(projects))
	if len(projects) == 1 {
		scope = "project archive written"
	}
	if encrypted {
		return "encrypted " + scope
	}
	return scope
}

func importProjectDetail(projects []string) string {
	if len(projects) == 1 {
		return "project archive imported"
	}
	return fmt.Sprintf("%d projects archive imported", len(projects))
}
