package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"continuum/internal/events"
	"continuum/internal/template"
)

func ContinuumPath() string {
	if path := os.Getenv("CONTINUUM_PATH"); path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, ".continuum")
	}
	return filepath.Join(home, ".continuum")
}

func ResolvePath(parts ...string) string {
	base := ContinuumPath()
	return filepath.Join(append([]string{base}, parts...)...)
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func ensureFile(path, content string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func writeFile(path string, content []byte, force bool) error {
	if force {
		return os.WriteFile(path, content, 0o644)
	}
	return ensureFile(path, string(content))
}

// initBase sets up the shared Continuum directory structure and writes
// profile.md and the agent skill file. Called by both Init and InitSession.
func initBase(base string, force bool) error {
	template.SetBasePath(base)

	dirs := []string{
		base,
		filepath.Join(base, "local"),
		filepath.Join(base, "exports"),
		filepath.Join(base, "skills"),
	}

	for _, dir := range dirs {
		if err := ensureDir(dir); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", dir, err)
		}
	}

	if err := template.InitTemplates(force); err != nil {
		return fmt.Errorf("cannot initialize templates: %w", err)
	}

	profileData, err := template.GetProfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot load profile template: %v\n", err)
	} else {
		if err := writeFile(filepath.Join(base, "profile.md"), profileData, force); err != nil {
			return fmt.Errorf("cannot create profile.md: %w", err)
		}
	}

	agentData, err := template.GetAgentSkill()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot load agent skill template: %v\n", err)
	} else {
		if err := writeFile(filepath.Join(base, "skills", "agent.md"), agentData, force); err != nil {
			return fmt.Errorf("cannot create agent skill: %w", err)
		}
	}

	targetsData, err := template.GetAgentTargets()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot load agent-targets template: %v\n", err)
	} else {
		dest := filepath.Join(base, "agent-targets.txt")
		if err := writeFile(dest, targetsData, force); err != nil {
			return fmt.Errorf("cannot create agent-targets.txt: %w", err)
		}
	}

	return nil
}

func Init(projectName string, force bool) error {
	base := ContinuumPath()

	if err := ValidateProjectName(projectName); err != nil {
		return err
	}

	if err := initBase(base, force); err != nil {
		return err
	}

	if err := ensureGitRepo(base, force); err != nil {
		return err
	}

	projectTasksDir := filepath.Join(base, "projects", projectName, "tasks")
	if err := ensureDir(projectTasksDir); err != nil {
		return fmt.Errorf("cannot create project directory: %w", err)
	}

	projectData, err := template.GetProject(projectName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot load project template: %v\n", err)
	} else {
		dest := filepath.Join(base, "projects", projectName, "project.md")
		if err := writeFile(dest, projectData, force); err != nil {
			return fmt.Errorf("cannot create project.md: %w", err)
		}
	}

	files, err := listTrackedFiles(base)
	if err != nil {
		return fmt.Errorf("cannot list files for project init commit: %w", err)
	}
	if err := events.Append(projectName, "", "project_initialized", "ok", "project initialized"); err == nil {
		files = append([]string{events.ActivityRelPath()}, files...)
	}
	if err := CommitFiles(
		fmt.Sprintf("init(%s): project initialized", projectName),
		files,
	); err != nil {
		return err
	}
	PushBestEffort()

	fmt.Printf("Continuum initialized for project '%s'\n", projectName)
	fmt.Println("Templates: .continuum/templates/")
	fmt.Println("Edit these files to customize defaults.")
	return nil
}

func InitSession(force bool) error {
	base := ContinuumPath()

	if err := initBase(base, force); err != nil {
		return err
	}

	if err := ensureGitRepo(base, force); err != nil {
		return err
	}

	if force {
		files, err := listTrackedFiles(base)
		if err != nil {
			return fmt.Errorf("cannot list files for session refresh commit: %w", err)
		}
		if err := events.Append("", "", "session_refreshed", "ok", "continuum refreshed"); err == nil {
			files = append([]string{events.ActivityRelPath()}, files...)
		}
		if err := CommitFiles("init: continuum refreshed", files); err != nil {
			return err
		}
	}
	PushBestEffort()

	fmt.Println("Session initialized.")
	fmt.Println("Templates: .continuum/templates/")
	fmt.Println("Run 'ctx project init <project>' to add a project.")
	return nil
}

func ProjectNameFromDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot get current directory: %w", err)
	}
	return filepath.Base(cwd), nil
}

func DetectProject() (string, error) {
	projectName, err := ProjectNameFromDir()
	if err != nil {
		return "", err
	}

	projectDir := filepath.Join(ContinuumPath(), "projects", projectName)

	if _, err := os.Stat(projectDir); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("project '%s' not found. Run 'ctx init' first", projectName)
		}
		return "", fmt.Errorf("cannot check project: %w", err)
	}

	return projectName, nil
}

func ListProjects() ([]string, error) {
	projectsDir := filepath.Join(ContinuumPath(), "projects")

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("cannot read projects directory: %w", err)
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() {
			projects = append(projects, entry.Name())
		}
	}

	sort.Strings(projects)
	return projects, nil
}
