package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"continuum/internal/template"
)

func withTempContinuum(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONTINUUM_PATH", dir)
	return dir
}

func TestContinuumPath_EnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONTINUUM_PATH", dir)
	if got := ContinuumPath(); got != dir {
		t.Errorf("ContinuumPath() = %q, want %q", got, dir)
	}
}

func TestValidateProjectName(t *testing.T) {
	valid := []string{"continuum", "my-project", "proj_1", "demo.v2"}
	for _, name := range valid {
		if err := ValidateProjectName(name); err != nil {
			t.Fatalf("ValidateProjectName(%q) unexpected error: %v", name, err)
		}
	}

	invalid := []string{"", ".", "..", "../evil", "bad/name", "bad\\name", "-flag", "white space", strings.Repeat("a", maxNameLength+1)}
	for _, name := range invalid {
		if err := ValidateProjectName(name); err == nil {
			t.Fatalf("ValidateProjectName(%q) expected error", name)
		}
	}
}

func TestInit_RejectsInvalidProjectName(t *testing.T) {
	withTempContinuum(t)
	if err := Init("../evil", false); err == nil {
		t.Fatal("expected Init to reject invalid project name")
	}
}

func TestListProjects_Empty(t *testing.T) {
	withTempContinuum(t)
	projects, err := ListProjects()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestListProjects(t *testing.T) {
	base := withTempContinuum(t)

	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := os.MkdirAll(filepath.Join(base, "projects", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	projects, err := ListProjects()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 3 {
		t.Errorf("expected 3 projects, got %d: %v", len(projects), projects)
	}
	// ListProjects returns sorted results
	if projects[0] != "alpha" || projects[1] != "beta" || projects[2] != "gamma" {
		t.Errorf("unexpected order: %v", projects)
	}
}

func TestListTrackedFiles_RespectsGitIgnore(t *testing.T) {
	base := withTempContinuum(t)

	if err := os.WriteFile(filepath.Join(base, ".gitignore"), []byte(".DS_Store\nThumbs.db\n._*\nignored/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", base, "init", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	for _, rel := range []string{
		".DS_Store",
		"Thumbs.db",
		"._project.md",
		"projects/.DS_Store",
		"projects/demo/Thumbs.db",
		"ignored/notes.md",
		"projects/demo/project.md",
		"agent-targets.txt",
	} {
		path := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := listTrackedFiles(base)
	if err != nil {
		t.Fatalf("listTrackedFiles: %v", err)
	}
	joined := strings.Join(files, "\n")
	for _, unwanted := range []string{".DS_Store", "Thumbs.db", "._project.md", "projects/.DS_Store", "projects/demo/Thumbs.db", "ignored/notes.md"} {
		if strings.Contains(joined, unwanted) {
			t.Fatalf("unexpected ignored file in tracked list: %s\nfiles:\n%s", unwanted, joined)
		}
	}
	for _, wanted := range []string{"agent-targets.txt", "projects/demo/project.md"} {
		if !strings.Contains(joined, wanted) {
			t.Fatalf("missing expected tracked file %s\nfiles:\n%s", wanted, joined)
		}
	}
}

func TestDetectProject_NotFound(t *testing.T) {
	withTempContinuum(t)
	_, err := DetectProject()
	if err == nil {
		t.Error("expected error for missing project, got nil")
	}
}

func TestDetectProject_Found(t *testing.T) {
	base := withTempContinuum(t)

	cwd, _ := os.Getwd()
	projectName := filepath.Base(cwd)

	if err := os.MkdirAll(filepath.Join(base, "projects", projectName), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := DetectProject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != projectName {
		t.Errorf("DetectProject() = %q, want %q", got, projectName)
	}
}

func writeTemplateSet(t *testing.T, dir, suffix string) {
	t.Helper()
	for _, file := range []struct {
		name    string
		content string
	}{
		{"profile.md", "profile " + suffix},
		{"project.md", "project comment %s\nproject name %s\n" + suffix},
		{"bootstrap.md", "bootstrap %s %s %s " + suffix},
		{"agent.md", "agent " + suffix},
		{"agent-targets.txt", "AGENTS.md\n"},
	} {
		if err := os.WriteFile(filepath.Join(dir, file.name), []byte(file.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInit_DoesNotOverwriteExistingProjectWithoutForce(t *testing.T) {
	base := withTempContinuum(t)
	source := t.TempDir()
	writeTemplateSet(t, source, "v1")
	template.SetSourcePath(source)
	t.Cleanup(func() { template.SetSourcePath("") })

	if err := Init("my-project", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	projectPath := filepath.Join(base, "projects", "my-project", "project.md")
	if err := os.WriteFile(projectPath, []byte("custom project"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Init("my-project", false); err != nil {
		t.Fatalf("second Init() error: %v", err)
	}

	data, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "custom project" {
		t.Fatalf("expected project.md to be preserved, got %q", string(data))
	}
}

func TestInit_ForceOverwritesExistingProject(t *testing.T) {
	base := withTempContinuum(t)
	source := t.TempDir()
	writeTemplateSet(t, source, "v2")
	template.SetSourcePath(source)
	t.Cleanup(func() { template.SetSourcePath("") })

	projectPath := filepath.Join(base, "projects", "my-project", "project.md")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, []byte("custom project"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Init("my-project", true); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	data, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Contains(content, "custom project") {
		t.Fatalf("expected force init to overwrite project.md, got %q", content)
	}
	if !strings.Contains(content, "project name my-project") {
		t.Fatalf("expected formatted project template, got %q", content)
	}
}

func TestInit_CreatesGitRepoAndInitialCommit(t *testing.T) {
	base := withTempContinuum(t)
	source := t.TempDir()
	writeTemplateSet(t, source, "git")
	template.SetSourcePath(source)
	t.Cleanup(func() { template.SetSourcePath("") })

	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(base, ".git")); err != nil {
		t.Fatalf("expected git repo to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, ".gitignore")); err != nil {
		t.Fatalf("expected .gitignore to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, ".gitattributes")); err != nil {
		t.Fatalf("expected .gitattributes to exist: %v", err)
	}
	gitignore, err := os.ReadFile(filepath.Join(base, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	for _, needle := range []string{".DS_Store", "Thumbs.db", "._*"} {
		if !strings.Contains(string(gitignore), needle) {
			t.Fatalf("expected .gitignore to contain %q, got:\n%s", needle, gitignore)
		}
	}
	gitattributes, err := os.ReadFile(filepath.Join(base, ".gitattributes"))
	if err != nil {
		t.Fatalf("read .gitattributes: %v", err)
	}
	if !strings.Contains(string(gitattributes), "events/activity.ndjson merge=union") {
		t.Fatalf("expected .gitattributes to configure merge=union for activity stream, got:\n%s", gitattributes)
	}

	out, err := exec.Command("git", "-C", base, "log", "--format=%s", "-1").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "init: continuum initialized" {
		t.Fatalf("unexpected initial commit message: %q", strings.TrimSpace(string(out)))
	}
}

func TestInit_ProjectCreatesGitCommit(t *testing.T) {
	base := withTempContinuum(t)
	source := t.TempDir()
	writeTemplateSet(t, source, "git")
	template.SetSourcePath(source)
	t.Cleanup(func() { template.SetSourcePath("") })

	if err := Init("my-project", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	out, err := exec.Command("git", "-C", base, "log", "--format=%s", "-1").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "init(my-project): project initialized" {
		t.Fatalf("unexpected project commit message: %q", strings.TrimSpace(string(out)))
	}
}

func TestInit_ForceLeavesGitWorktreeClean(t *testing.T) {
	base := withTempContinuum(t)
	source := t.TempDir()
	writeTemplateSet(t, source, "force")
	template.SetSourcePath(source)
	t.Cleanup(func() { template.SetSourcePath("") })

	if err := Init("my-project", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	updated := t.TempDir()
	writeTemplateSet(t, updated, "force-updated")
	template.SetSourcePath(updated)

	if err := Init("my-project", true); err != nil {
		t.Fatalf("Init(..., true) error: %v", err)
	}

	out, err := exec.Command("git", "-C", base, "status", "--short").CombinedOutput()
	if err != nil {
		t.Fatalf("git status failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("expected clean worktree after force init, got:\n%s", out)
	}
}

func TestDeleteProject_RestoresProjectOnStageFailure(t *testing.T) {
	base := withTempContinuum(t)
	source := t.TempDir()
	writeTemplateSet(t, source, "delete")
	template.SetSourcePath(source)
	t.Cleanup(func() { template.SetSourcePath("") })

	if err := Init("my-project", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	lockPath := filepath.Join(base, ".git", "index.lock")
	if err := os.WriteFile(lockPath, []byte("locked"), 0o644); err != nil {
		t.Fatalf("write index.lock: %v", err)
	}
	defer os.Remove(lockPath)

	err := DeleteProject("my-project")
	if err == nil {
		t.Fatal("expected DeleteProject to fail when git index is locked")
	}

	if _, statErr := os.Stat(filepath.Join(base, "projects", "my-project")); statErr != nil {
		t.Fatalf("expected project directory to be restored, got: %v", statErr)
	}
}

func TestOnboardProject_WritesContentAndCommits(t *testing.T) {
	base := withTempContinuum(t)
	source := t.TempDir()
	writeTemplateSet(t, source, "onboard")
	template.SetSourcePath(source)
	t.Cleanup(func() { template.SetSourcePath("") })

	content := []byte("# PROJECT\n\n## Name\nmy-project\n\n## Summary\nReal project context\n")
	if err := OnboardProject("my-project", content, false); err != nil {
		t.Fatalf("OnboardProject() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, "projects", "my-project", "project.md"))
	if err != nil {
		t.Fatalf("read onboarded project: %v", err)
	}
	if strings.TrimSpace(string(data)) != strings.TrimSpace(string(content)) {
		t.Fatalf("unexpected project.md content:\n%s", data)
	}

	out, err := exec.Command("git", "-C", base, "log", "--format=%s", "-1").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "onboard(my-project): project context updated" {
		t.Fatalf("unexpected onboarding commit message: %q", strings.TrimSpace(string(out)))
	}
}

func TestOnboardProject_RejectsOverwriteWithoutForce(t *testing.T) {
	base := withTempContinuum(t)
	source := t.TempDir()
	writeTemplateSet(t, source, "onboard")
	template.SetSourcePath(source)
	t.Cleanup(func() { template.SetSourcePath("") })

	projectPath := filepath.Join(base, "projects", "my-project", "project.md")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, []byte("# PROJECT\n\n## Summary\nExisting real context\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := OnboardProject("my-project", []byte("# PROJECT\n\n## Summary\nReplacement context\n"), false)
	if err == nil {
		t.Fatal("expected OnboardProject to reject overwriting real content without force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected force hint, got %v", err)
	}
}

func TestOnboardProject_OverwritesTemplateWithoutForce(t *testing.T) {
	base := withTempContinuum(t)
	source := t.TempDir()
	writeTemplateSet(t, source, "onboard")
	template.SetSourcePath(source)
	t.Cleanup(func() { template.SetSourcePath("") })

	if err := Init("my-project", false); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	content := []byte("# PROJECT\n\n## Name\nmy-project\n\n## Summary\nOnboarded context\n")
	if err := OnboardProject("my-project", content, false); err != nil {
		t.Fatalf("OnboardProject() over template error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(base, "projects", "my-project", "project.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != strings.TrimSpace(string(content)) {
		t.Fatalf("expected template content to be replaced, got:\n%s", data)
	}
}

func TestOnboardProject_ForceOverwritesExistingContent(t *testing.T) {
	base := withTempContinuum(t)
	source := t.TempDir()
	writeTemplateSet(t, source, "onboard")
	template.SetSourcePath(source)
	t.Cleanup(func() { template.SetSourcePath("") })

	projectPath := filepath.Join(base, "projects", "my-project", "project.md")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, []byte("# PROJECT\n\n## Summary\nExisting real context\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	content := []byte("# PROJECT\n\n## Summary\nReplacement context\n")
	if err := OnboardProject("my-project", content, true); err != nil {
		t.Fatalf("OnboardProject(..., true) error: %v", err)
	}

	data, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != strings.TrimSpace(string(content)) {
		t.Fatalf("expected content to be overwritten, got:\n%s", data)
	}
}
