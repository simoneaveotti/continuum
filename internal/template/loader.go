package template

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var basePath string
var sourcePath string
var executablePath = os.Executable

const BootstrapVersion = "2026-05-06.1"

func SetBasePath(path string) {
	basePath = path
}

func SetSourcePath(path string) {
	sourcePath = path
}

func ValidateSourcePath(path string) error {
	required := []string{"profile.md", "project.md", "bootstrap.md", "agent.md", "agent-targets.txt"}
	for _, name := range required {
		fullPath := filepath.Join(path, name)
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("template source missing %s", name)
			}
			return fmt.Errorf("cannot access template source %s: %w", fullPath, err)
		}
		if info.IsDir() {
			return fmt.Errorf("template source entry is a directory: %s", fullPath)
		}
	}
	return nil
}

func userTemplatesDir() string {
	if basePath == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = os.Getenv("HOME")
		}
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		if home == "" {
			home = "."
		}
		basePath = filepath.Join(home, ".ctx")
	}
	return filepath.Join(basePath, "templates")
}

func repoTemplatesDir() string {
	execPath, err := executablePath()
	if err != nil {
		return "templates"
	}
	return filepath.Join(filepath.Dir(execPath), "templates")
}

func installedTemplatesDir() string {
	execPath, err := executablePath()
	if err != nil {
		return ""
	}
	return filepath.Clean(filepath.Join(filepath.Dir(execPath), "..", "share", "continuum", "templates"))
}

func sourceTemplatesDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "templates")
}

func workingDirTemplatesDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(cwd, "templates")
}

func templateSearchPaths(name string) []string {
	dirs := []string{userTemplatesDir()}
	if sourcePath != "" {
		dirs = append(dirs, sourcePath)
	} else {
		dirs = append(dirs, installedTemplatesDir(), repoTemplatesDir(), sourceTemplatesDir(), workingDirTemplatesDir())
	}

	var paths []string
	for _, dir := range dirs {
		if dir != "" {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	return paths
}

func initTemplateSearchPaths(name string) []string {
	var dirs []string
	if sourcePath != "" {
		dirs = append(dirs, sourcePath)
	} else {
		dirs = append(dirs, installedTemplatesDir(), repoTemplatesDir(), sourceTemplatesDir(), workingDirTemplatesDir())
	}

	var paths []string
	for _, dir := range dirs {
		if dir != "" {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	return paths
}

func readFirstTemplate(name string, paths []string) ([]byte, error) {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("cannot read %s: %w", name, err)
		}
	}

	return nil, fmt.Errorf("template not found: %s", name)
}

func findTemplate(name string) ([]byte, error) {
	return readFirstTemplate(name, templateSearchPaths(name))
}

func GetProfile() ([]byte, error) {
	return findTemplate("profile.md")
}

func GetProject(project string) ([]byte, error) {
	data, err := findTemplate("project.md")
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil, string(data), project, project), nil
}

func GetBootstrap(project string) ([]byte, error) {
	data, err := findTemplate("bootstrap.md")
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil, string(data), project, BootstrapVersion), nil
}

func GetAgentSkill() ([]byte, error) {
	return findTemplate("agent.md")
}

func GetAgentTargets() ([]byte, error) {
	return findTemplate("agent-targets.txt")
}

func InitTemplates(force bool) error {
	if err := os.MkdirAll(userTemplatesDir(), 0o755); err != nil {
		return err
	}

	files := []string{"profile.md", "project.md", "bootstrap.md", "agent.md", "agent-targets.txt"}

	for _, name := range files {
		path := filepath.Join(userTemplatesDir(), name)
		_, err := os.Stat(path)
		if err == nil && !force {
			continue
		}
		if err != nil && !os.IsNotExist(err) {
			continue
		}

		data, err := readFirstTemplate(name, initTemplateSearchPaths(name))
		if err != nil {
			continue
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("cannot write %s: %w", name, err)
		}
	}

	return nil
}
