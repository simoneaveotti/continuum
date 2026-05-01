package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempBase(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	SetBasePath(dir)
	SetSourcePath("")
	t.Cleanup(func() {
		SetBasePath("")
		SetSourcePath("")
	})

	// Copy real templates into the temp base so findTemplate can locate them
	templatesDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	repoTemplates := repoTemplatesDir()
	for _, name := range []string{"profile.md", "project.md", "bootstrap.md", "agent.md", "agent-targets.txt"} {
		data, err := os.ReadFile(filepath.Join(repoTemplates, name))
		if err != nil {
			continue // skip if not found in repo dir (CI)
		}
		os.WriteFile(filepath.Join(templatesDir, name), data, 0o644)
	}

	return dir
}

func TestFindTemplate_NotFound(t *testing.T) {
	dir := t.TempDir()
	SetBasePath(dir)
	t.Cleanup(func() { SetBasePath("") })

	_, err := findTemplate("nonexistent.md")
	if err == nil {
		t.Error("expected error for missing template, got nil")
	}
}

func TestFindTemplate_Found(t *testing.T) {
	dir := t.TempDir()
	SetBasePath(dir)
	t.Cleanup(func() { SetBasePath("") })

	templatesDir := filepath.Join(dir, "templates")
	os.MkdirAll(templatesDir, 0o755)
	os.WriteFile(filepath.Join(templatesDir, "test.md"), []byte("hello"), 0o644)

	data, err := findTemplate("test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("got %q, want %q", string(data), "hello")
	}
}

func TestGetBootstrap_SubstitutesProject(t *testing.T) {
	dir := t.TempDir()
	SetBasePath(dir)
	t.Cleanup(func() { SetBasePath("") })

	templatesDir := filepath.Join(dir, "templates")
	os.MkdirAll(templatesDir, 0o755)
	os.WriteFile(filepath.Join(templatesDir, "bootstrap.md"), []byte("project: %[1]s\ncmd: --project=%[1]s\nlist: --project=%[1]s\ncheck: --project=%[1]s"), 0o644)

	data, err := GetBootstrap("myproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "project: myproject") {
		t.Errorf("first %%s not substituted: %q", out)
	}
	if !strings.Contains(out, "--project=myproject") {
		t.Errorf("second/third %%s not substituted: %q", out)
	}
	if strings.Count(out, "myproject") != 4 {
		t.Errorf("expected four substituted placeholders, got %q", out)
	}
}

func TestBootstrapRetainsCriticalContinuumProtocol(t *testing.T) {
	dir := t.TempDir()
	SetBasePath(dir)
	SetSourcePath("")
	t.Cleanup(func() {
		SetBasePath("")
		SetSourcePath("")
	})

	data, err := findTemplate("bootstrap.md")
	if err != nil {
		t.Fatalf("findTemplate(bootstrap.md): %v", err)
	}

	out := string(data)
	for _, needle := range []string{
		"## State 0: Session Start (CRITICAL)",
		"ctx context --project=%[1]s",
		"## State 1: Task Check (CRITICAL)",
		"ctx list --project=%[1]s",
		"CONTINUUM_AGENT=<stable-name> ctx task start <task> --project=%[1]s",
		"CONTINUUM_AGENT=<stable-name> ctx capture <task> --project=%[1]s --type=state --yes",
		"--type=proposal",
		"CONTINUUM_AGENT=<stable-name> ctx handoff <task> --project=%[1]s --yes",
		"CONTINUUM_AGENT=<stable-name> ctx task close <task> --project=%[1]s",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("bootstrap missing critical protocol text %q", needle)
		}
	}
}

func TestGetProject_SubstitutesProject(t *testing.T) {
	dir := t.TempDir()
	SetBasePath(dir)
	SetSourcePath("")
	t.Cleanup(func() {
		SetBasePath("")
		SetSourcePath("")
	})

	templatesDir := filepath.Join(dir, "templates")
	os.MkdirAll(templatesDir, 0o755)
	os.WriteFile(filepath.Join(templatesDir, "project.md"), []byte("comment %s\nname %s"), 0o644)

	data, err := GetProject("travel-manager")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "comment travel-manager") {
		t.Errorf("first %%s not substituted: %q", out)
	}
	if !strings.Contains(out, "name travel-manager") {
		t.Errorf("second %%s not substituted: %q", out)
	}
}

func TestInitTemplates(t *testing.T) {
	dir := t.TempDir()
	SetBasePath(dir)
	SetSourcePath("")
	t.Cleanup(func() {
		SetBasePath("")
		SetSourcePath("")
	})

	// Seed repo-side templates so InitTemplates can copy them
	repoDir := filepath.Join(dir, "repo-templates")
	os.MkdirAll(repoDir, 0o755)
	for _, name := range []string{"profile.md", "project.md", "bootstrap.md", "agent.md", "agent-targets.txt"} {
		os.WriteFile(filepath.Join(repoDir, name), []byte("# "+name), 0o644)
	}

	// Point repoTemplatesDir at our fake repo dir by writing files to userTemplatesDir instead
	userDir := filepath.Join(dir, "templates")
	os.MkdirAll(userDir, 0o755)
	for _, name := range []string{"profile.md", "project.md", "bootstrap.md", "agent.md", "agent-targets.txt"} {
		os.WriteFile(filepath.Join(userDir, name), []byte("# "+name), 0o644)
	}

	if err := InitTemplates(false); err != nil {
		t.Fatalf("InitTemplates() error: %v", err)
	}

	for _, name := range []string{"profile.md", "project.md", "bootstrap.md", "agent.md", "agent-targets.txt"} {
		if _, err := os.Stat(filepath.Join(userDir, name)); err != nil {
			t.Errorf("expected %s to exist after InitTemplates: %v", name, err)
		}
	}
}

func TestFindTemplate_UsesExplicitSourcePath(t *testing.T) {
	dir := t.TempDir()
	SetBasePath(filepath.Join(dir, "continuum"))
	SetSourcePath(filepath.Join(dir, "source"))
	t.Cleanup(func() {
		SetBasePath("")
		SetSourcePath("")
	})

	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourcePath, "profile.md"), []byte("from-source"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := findTemplate("profile.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "from-source" {
		t.Fatalf("expected explicit source content, got %q", string(data))
	}
}

func TestValidateSourcePath(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"profile.md", "project.md", "bootstrap.md", "agent.md", "agent-targets.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := ValidateSourcePath(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSourcePath_MissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "profile.md"), []byte("# profile"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ValidateSourcePath(dir); err == nil {
		t.Fatal("expected error for incomplete template source")
	}
}

func TestInitTemplates_ForceUsesExplicitSourceInsteadOfExistingUserTemplates(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	userDir := filepath.Join(dir, "continuum", "templates")

	SetBasePath(filepath.Join(dir, "continuum"))
	SetSourcePath(sourceDir)
	t.Cleanup(func() {
		SetBasePath("")
		SetSourcePath("")
	})

	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"profile.md", "project.md", "bootstrap.md", "agent.md", "agent-targets.txt"} {
		if err := os.WriteFile(filepath.Join(sourceDir, name), []byte("new "+name), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(userDir, name), []byte("old "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := InitTemplates(true); err != nil {
		t.Fatalf("InitTemplates(true) error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(userDir, "bootstrap.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new bootstrap.md" {
		t.Fatalf("expected explicit source to overwrite user template, got %q", string(data))
	}
}
