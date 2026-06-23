package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Index tests ---

func TestParseIndex_Empty(t *testing.T) {
	entries := ParseIndex("# Skills Index\n")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseIndex_SingleEntryWithDescription(t *testing.T) {
	content := "# Skills Index\n\n## auth-patterns\nLaravel Passport + LDAP hybrid auth.\n"
	entries := ParseIndex(content)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "auth-patterns" {
		t.Errorf("name = %q", entries[0].Name)
	}
	if entries[0].Description != "Laravel Passport + LDAP hybrid auth." {
		t.Errorf("description = %q", entries[0].Description)
	}
}

func TestParseIndex_EntryWithoutDescription(t *testing.T) {
	content := "# Skills Index\n\n## no-desc\n"
	entries := ParseIndex(content)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Description != "" {
		t.Errorf("expected empty description, got %q", entries[0].Description)
	}
}

func TestParseIndex_MultipleEntries(t *testing.T) {
	content := "# Skills Index\n\n## alpha\nFirst skill.\n\n## beta\nSecond skill.\n"
	entries := ParseIndex(content)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "alpha" || entries[1].Name != "beta" {
		t.Errorf("unexpected names: %v", entries)
	}
}

func TestRenderIndex_Empty(t *testing.T) {
	got := RenderIndex(nil)
	if !strings.HasPrefix(got, "# Skills Index") {
		t.Errorf("expected Skills Index header, got: %q", got)
	}
}

func TestRenderIndex_SortedOutput(t *testing.T) {
	entries := []IndexEntry{
		{Name: "zebra", Description: "last"},
		{Name: "alpha", Description: "first"},
	}
	got := RenderIndex(entries)
	alphaPos := strings.Index(got, "## alpha")
	zebraPos := strings.Index(got, "## zebra")
	if alphaPos == -1 || zebraPos == -1 || alphaPos > zebraPos {
		t.Errorf("entries not sorted alphabetically:\n%s", got)
	}
}

func TestRenderIndex_RoundTrip(t *testing.T) {
	entries := []IndexEntry{
		{Name: "auth-patterns", Description: "Laravel auth patterns."},
		{Name: "db-safety", Description: ""},
	}
	rendered := RenderIndex(entries)
	parsed := ParseIndex(rendered)
	if len(parsed) != 2 {
		t.Fatalf("expected 2 entries after roundtrip, got %d", len(parsed))
	}
	if parsed[0].Name != "auth-patterns" || parsed[0].Description != "Laravel auth patterns." {
		t.Errorf("first entry mismatch: %+v", parsed[0])
	}
	if parsed[1].Name != "db-safety" || parsed[1].Description != "" {
		t.Errorf("second entry mismatch: %+v", parsed[1])
	}
}

func TestUpdateIndex_AddsNewEntry(t *testing.T) {
	base := skillsDir(t)
	if err := UpdateIndex(base, "auth-patterns", "Laravel auth."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(base, "index.md"))
	if !strings.Contains(string(data), "## auth-patterns") {
		t.Errorf("index missing entry:\n%s", data)
	}
	if !strings.Contains(string(data), "Laravel auth.") {
		t.Errorf("index missing description:\n%s", data)
	}
}

func TestUpdateIndex_UpdatesExistingDescription(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "index.md"), []byte("# Skills Index\n\n## auth-patterns\nOld desc.\n"), 0o644)

	if err := UpdateIndex(base, "auth-patterns", "New desc."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(base, "index.md"))
	if strings.Contains(string(data), "Old desc.") {
		t.Errorf("index still has old description:\n%s", data)
	}
	if !strings.Contains(string(data), "New desc.") {
		t.Errorf("index missing new description:\n%s", data)
	}
}

func TestUpdateIndex_KeepsExistingDescriptionWhenNoneProvided(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "index.md"), []byte("# Skills Index\n\n## auth-patterns\nExisting desc.\n"), 0o644)

	if err := UpdateIndex(base, "auth-patterns", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(base, "index.md"))
	if !strings.Contains(string(data), "Existing desc.") {
		t.Errorf("existing description was lost:\n%s", data)
	}
}

func TestUpdateIndex_CreatesIndexIfMissing(t *testing.T) {
	base := skillsDir(t)
	if err := UpdateIndex(base, "new-skill", "A new skill."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "index.md")); err != nil {
		t.Errorf("index.md not created: %v", err)
	}
}

func TestUpdateIndex_SortsEntries(t *testing.T) {
	base := skillsDir(t)
	UpdateIndex(base, "zebra", "last")
	UpdateIndex(base, "alpha", "first")

	data, _ := os.ReadFile(filepath.Join(base, "index.md"))
	alphaPos := strings.Index(string(data), "## alpha")
	zebraPos := strings.Index(string(data), "## zebra")
	if alphaPos == -1 || zebraPos == -1 || alphaPos > zebraPos {
		t.Errorf("entries not sorted:\n%s", data)
	}
}

func TestListWithDescriptions_ReadsIndex(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "index.md"), []byte("# Skills Index\n\n## auth-patterns\nLaravel auth.\n"), 0o644)

	entries, fromIndex, err := ListWithDescriptions(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fromIndex {
		t.Error("expected fromIndex=true")
	}
	if len(entries) != 1 || entries[0].Name != "auth-patterns" {
		t.Errorf("unexpected entries: %v", entries)
	}
}

func TestListWithDescriptions_FallbackToDirListing(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "orphan.md"), []byte("content"), 0o644)

	entries, fromIndex, err := ListWithDescriptions(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fromIndex {
		t.Error("expected fromIndex=false when index.md absent")
	}
	if len(entries) != 1 || entries[0].Name != "orphan" {
		t.Errorf("unexpected entries: %v", entries)
	}
}

func TestMigrateAgentToIndex_CreatesIndexIfAgentExists(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "agent.md"), []byte("old agent content"), 0o644)

	migrated, err := MigrateAgentToIndex(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !migrated {
		t.Error("expected migration to happen")
	}
	if _, err := os.Stat(filepath.Join(base, "index.md")); err != nil {
		t.Errorf("index.md not created: %v", err)
	}
	// agent.md must still exist (not deleted)
	if _, err := os.Stat(filepath.Join(base, "agent.md")); err != nil {
		t.Errorf("agent.md should not be deleted: %v", err)
	}
	// index content must NOT contain old agent content
	data, _ := os.ReadFile(filepath.Join(base, "index.md"))
	if strings.Contains(string(data), "old agent content") {
		t.Error("index.md should not contain agent.md content")
	}
}

func TestMigrateAgentToIndex_NoOpIfAgentMissing(t *testing.T) {
	base := skillsDir(t)

	migrated, err := MigrateAgentToIndex(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if migrated {
		t.Error("expected no migration when agent.md absent")
	}
}

func TestMigrateAgentToIndex_NoOpIfIndexAlreadyExists(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "agent.md"), []byte("old"), 0o644)
	existingIndex := "# Skills Index\n\n## existing\nKeep me.\n"
	os.WriteFile(filepath.Join(base, "index.md"), []byte(existingIndex), 0o644)

	migrated, err := MigrateAgentToIndex(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if migrated {
		t.Error("expected no migration when index.md already exists")
	}
	data, _ := os.ReadFile(filepath.Join(base, "index.md"))
	if string(data) != existingIndex {
		t.Error("existing index.md should not be modified")
	}
}

func TestDelete_FlatFile(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "auth-patterns.md"), []byte("content"), 0o644)

	if err := Delete(base, "auth-patterns"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "auth-patterns.md")); !os.IsNotExist(err) {
		t.Error("skill file should have been deleted")
	}
}

func TestDelete_DirectoryLayout(t *testing.T) {
	base := skillsDir(t)
	subDir := filepath.Join(base, "db-safety")
	os.MkdirAll(subDir, 0o755)
	os.WriteFile(filepath.Join(subDir, "SKILL.md"), []byte("content"), 0o644)

	if err := Delete(base, "db-safety"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(subDir); !os.IsNotExist(err) {
		t.Error("skill directory should have been deleted")
	}
}

func TestDelete_NotFound(t *testing.T) {
	base := skillsDir(t)

	err := Delete(base, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing skill, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention skill name: %v", err)
	}
}

func TestDelete_RemovesEntryFromIndex(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "auth-patterns.md"), []byte("content"), 0o644)
	os.WriteFile(filepath.Join(base, "index.md"), []byte("# Skills Index\n\n## auth-patterns\nDesc.\n\n## other\nOther.\n"), 0o644)

	if err := Delete(base, "auth-patterns"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(base, "index.md"))
	if strings.Contains(string(data), "auth-patterns") {
		t.Errorf("index still contains deleted skill:\n%s", data)
	}
	if !strings.Contains(string(data), "other") {
		t.Errorf("index lost unrelated entry:\n%s", data)
	}
}

func TestDelete_NoIndexIsSilentlySkipped(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "auth-patterns.md"), []byte("content"), 0o644)

	if err := Delete(base, "auth-patterns"); err != nil {
		t.Fatalf("unexpected error when index missing: %v", err)
	}
}

func skillsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONTINUUM_PATH", dir)
	skillsPath := filepath.Join(dir, "skills")
	if err := os.MkdirAll(skillsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	return skillsPath
}

func TestList_Empty(t *testing.T) {
	base := skillsDir(t)
	names, err := List(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 skills, got %d: %v", len(names), names)
	}
}

func TestList_FlatFile(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "auth-patterns.md"), []byte("# Auth"), 0o644)
	os.WriteFile(filepath.Join(base, "queue-gotchas.md"), []byte("# Queue"), 0o644)

	names, err := List(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 skills, got %d: %v", len(names), names)
	}
}

func TestList_DirectoryLayout(t *testing.T) {
	base := skillsDir(t)
	subDir := filepath.Join(base, "db-migration-safety")
	os.MkdirAll(subDir, 0o755)
	os.WriteFile(filepath.Join(subDir, "SKILL.md"), []byte("# DB"), 0o644)

	names, err := List(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "db-migration-safety" {
		t.Fatalf("expected [db-migration-safety], got %v", names)
	}
}

func TestList_MixedLayouts(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "flat-skill.md"), []byte("flat"), 0o644)
	subDir := filepath.Join(base, "dir-skill")
	os.MkdirAll(subDir, 0o755)
	os.WriteFile(filepath.Join(subDir, "SKILL.md"), []byte("dir"), 0o644)
	// Directory without SKILL.md should be ignored
	os.MkdirAll(filepath.Join(base, "no-skill-file"), 0o755)

	names, err := List(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 skills, got %d: %v", len(names), names)
	}
}

func TestList_IgnoresNonMdFiles(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "README.txt"), []byte("ignore me"), 0o644)
	os.WriteFile(filepath.Join(base, "valid.md"), []byte("valid"), 0o644)

	names, err := List(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "valid" {
		t.Fatalf("expected [valid], got %v", names)
	}
}

func TestList_SkillsDirMissing(t *testing.T) {
	names, err := List("/nonexistent/path/skills")
	if err != nil {
		t.Fatalf("missing skills dir should return empty, not error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 skills, got %d", len(names))
	}
}

func TestShow_FlatFile(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "auth-patterns.md"), []byte("# Auth Patterns"), 0o644)

	content, err := Show(base, "auth-patterns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "# Auth Patterns" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestShow_DirectoryLayout(t *testing.T) {
	base := skillsDir(t)
	subDir := filepath.Join(base, "db-safety")
	os.MkdirAll(subDir, 0o755)
	os.WriteFile(filepath.Join(subDir, "SKILL.md"), []byte("# DB Safety"), 0o644)

	content, err := Show(base, "db-safety")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "# DB Safety" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestShow_NotFound(t *testing.T) {
	base := skillsDir(t)

	_, err := Show(base, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing skill, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention skill name: %v", err)
	}
}

func TestShow_FlatFilePrecedesDirectory(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "ambiguous.md"), []byte("flat"), 0o644)
	subDir := filepath.Join(base, "ambiguous")
	os.MkdirAll(subDir, 0o755)
	os.WriteFile(filepath.Join(subDir, "SKILL.md"), []byte("dir"), 0o644)

	content, err := Show(base, "ambiguous")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "flat" {
		t.Errorf("expected flat file to take precedence, got: %q", content)
	}
}

func TestSave_WritesNewFile(t *testing.T) {
	base := skillsDir(t)

	err := Save(base, "new-skill", "# New Skill\ncontent", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(base, "new-skill.md"))
	if string(got) != "# New Skill\ncontent" {
		t.Errorf("unexpected content: %q", string(got))
	}
}

func TestSave_FailsIfExistsWithoutForce(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "existing.md"), []byte("old"), 0o644)

	err := Save(base, "existing", "new content", false)
	if err == nil {
		t.Fatal("expected error when overwriting without force, got nil")
	}
}

func TestSave_OverwritesWithForce(t *testing.T) {
	base := skillsDir(t)
	os.WriteFile(filepath.Join(base, "existing.md"), []byte("old"), 0o644)

	err := Save(base, "existing", "new content", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(base, "existing.md"))
	if string(got) != "new content" {
		t.Errorf("expected new content, got: %q", string(got))
	}
}

func TestSave_CreatesDirIfMissing(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "skills") // does not exist yet

	err := Save(base, "new-skill", "# content", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(base, "new-skill.md"))
	if string(got) != "# content" {
		t.Errorf("unexpected content: %q", string(got))
	}
}
