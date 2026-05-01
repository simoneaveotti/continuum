package export

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"continuum/internal/events"
	"continuum/internal/setup"
)

// makeZip builds an in-memory zip containing a single file with the given name and content.
func makeZip(t *testing.T, name, content string) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte(content))
	w.Close()
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func readActivityEvents(t *testing.T) []events.Event {
	t.Helper()
	items, _, err := events.ReadFromOffset(0)
	if err != nil {
		t.Fatalf("ReadFromOffset: %v", err)
	}
	return items
}

// --- extractZipFile ---

func TestExtractZipFile_Normal(t *testing.T) {
	dir := t.TempDir()
	zr := makeZip(t, "snapshot.md", "hello")

	if err := extractZipFile(zr.File[0], dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "snapshot.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("file content = %q, want %q", string(data), "hello")
	}
}

func TestExtractZipFile_ZipSlip(t *testing.T) {
	dir := t.TempDir()
	// A malicious zip entry trying to escape destDir
	zr := makeZip(t, "../../evil.txt", "malicious")

	err := extractZipFile(zr.File[0], dir)
	if err == nil {
		t.Error("expected error for zip slip path, got nil")
	}

	// Verify the file was NOT written outside the dir
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(dir), "evil.txt")); statErr == nil {
		t.Error("zip slip file was written outside destDir")
	}
}

func TestExtractZipFile_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	// Absolute path in zip entry
	zr := makeZip(t, "/etc/passwd", "malicious")

	err := extractZipFile(zr.File[0], dir)
	if err == nil {
		t.Error("expected error for absolute path in zip entry, got nil")
	}
}

// --- extractTaskName ---

func TestExtractTaskName_FromSnapshot(t *testing.T) {
	content := "# TASK SNAPSHOT\n\n## Task\nmy-feature\n\n## Objective\nDo something\n"
	zr := makeZip(t, "snapshot.md", content)

	got := extractTaskName("any.zip", zr)
	if got != "my-feature" {
		t.Errorf("extractTaskName = %q, want %q", got, "my-feature")
	}
}

func TestExtractTaskName_FallbackToFilename(t *testing.T) {
	// Snapshot with no ## Task section
	zr := makeZip(t, "snapshot.md", "# TASK SNAPSHOT\n\n## Objective\nDo something\n")

	got := extractTaskName("my-feature-share-20260101.zip", zr)
	if got != "my-feature-20260101" {
		t.Errorf("extractTaskName = %q, want %q", got, "my-feature-20260101")
	}
}

func TestExtractTaskName_EmptyZip(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	w.Close()
	zr, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

	got := extractTaskName("my-task.zip", zr)
	if got != "my-task" {
		t.Errorf("extractTaskName = %q, want %q", got, "my-task")
	}
}

// --- resolveOutputPath ---

func TestResolveOutputPath_DefaultDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONTINUUM_PATH", dir)

	path, err := resolveOutputPath("", "mytask", "md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Ext(path) != ".md" {
		t.Errorf("expected .md extension, got %q", path)
	}
}

func TestResolveOutputPath_CustomDir(t *testing.T) {
	dir := t.TempDir()

	path, err := resolveOutputPath(dir, "mytask", "md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Dir(path) != dir {
		t.Errorf("expected file in %q, got %q", dir, path)
	}
}

func TestResolveOutputPath_CustomFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.md")

	path, err := resolveOutputPath(target, "mytask", "md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != target {
		t.Errorf("expected %q, got %q", target, path)
	}
}

func TestExportAndImportTask_RestoresProjectMetadata(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	if err := setup.Init("demo", false); err != nil {
		t.Fatalf("setup.Init: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "demo", "tasks", "sample")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	snapshot := `# TASK SNAPSHOT

## Task
sample

## Project
demo

## Objective
Roundtrip test
`
	if err := os.WriteFile(filepath.Join(taskDir, "snapshot.test.md"), []byte(snapshot), 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "notes.md"), []byte("notes"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "sample.zip")
	output, err := ExportTask("sample", "demo", archivePath)
	if err != nil {
		t.Fatalf("ExportTask: %v", err)
	}

	importBase := t.TempDir()
	t.Setenv("CONTINUUM_PATH", importBase)
	if err := setup.InitSession(false); err != nil {
		t.Fatalf("setup.InitSession: %v", err)
	}

	taskName, err := ImportArchive(output, false, "")
	if err != nil {
		t.Fatalf("ImportArchive: %v", err)
	}
	if taskName != "sample" {
		t.Fatalf("imported task = %q", taskName)
	}

	projectPath := filepath.Join(importBase, "projects", "demo", "project.md")
	if _, err := os.Stat(projectPath); err != nil {
		t.Fatalf("expected imported project metadata, got: %v", err)
	}
}

func TestExportProjectsAndImport_RestoresProjectTree(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	if err := setup.Init("demo", false); err != nil {
		t.Fatalf("setup.Init: %v", err)
	}

	taskDir := filepath.Join(base, "projects", "demo", "tasks", "sample")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	snapshot := `# TASK SNAPSHOT

## Task
sample

## Project
demo

## Objective
Roundtrip export test
`
	if err := os.WriteFile(filepath.Join(taskDir, "snapshot.test.md"), []byte(snapshot), 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "notes.md"), []byte("notes"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "demo-project.zip")
	output, err := ExportProjects([]string{"demo"}, archivePath)
	if err != nil {
		t.Fatalf("ExportProjects: %v", err)
	}

	importBase := t.TempDir()
	t.Setenv("CONTINUUM_PATH", importBase)
	if err := setup.InitSession(false); err != nil {
		t.Fatalf("setup.InitSession: %v", err)
	}

	target, err := ImportArchive(output, false, "")
	if err != nil {
		t.Fatalf("ImportArchive: %v", err)
	}
	if target != "project demo" {
		t.Fatalf("imported target = %q", target)
	}

	for _, path := range []string{
		filepath.Join(importBase, "projects", "demo", "project.md"),
		filepath.Join(importBase, "projects", "demo", "tasks", "sample", "notes.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected imported file %s, got: %v", path, err)
		}
	}
}

func TestExportAndImportEmitActivityEvents(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)

	if err := setup.Init("demo", false); err != nil {
		t.Fatalf("setup.Init: %v", err)
	}
	taskDir := filepath.Join(base, "projects", "demo", "tasks", "sample")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	snapshot := "# TASK SNAPSHOT\n\n## Task\nsample\n\n## Project\ndemo\n"
	if err := os.WriteFile(filepath.Join(taskDir, "snapshot.test.md"), []byte(snapshot), 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "notes.md"), []byte("notes"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "sample.zip")
	output, err := ExportTask("sample", "demo", archivePath)
	if err != nil {
		t.Fatalf("ExportTask: %v", err)
	}
	if _, err := ImportArchive(output, false, ""); err != nil {
		t.Fatalf("ImportArchive: %v", err)
	}

	items := readActivityEvents(t)
	if len(items) < 2 {
		t.Fatalf("expected export/import events, got %d", len(items))
	}
	if items[len(items)-2].Type != "export" || items[len(items)-1].Type != "import" {
		t.Fatalf("unexpected trailing events: %#v", items[len(items)-2:])
	}
}

func TestEncryptDecryptAESGCMV2Roundtrip(t *testing.T) {
	plaintext := []byte("continuum secret")

	ciphertext, err := encryptData(plaintext, "correct horse battery staple", "")
	if err != nil {
		t.Fatalf("encryptData: %v", err)
	}
	if !bytes.HasPrefix(ciphertext, []byte(v2Magic)) {
		t.Fatalf("expected v2 magic prefix")
	}

	got, err := decryptData(ciphertext, "correct horse battery staple", "")
	if err != nil {
		t.Fatalf("decryptData: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("decryptData = %q, want %q", string(got), string(plaintext))
	}
}

func TestDecryptDefaultRejectsNonV2Ciphertext(t *testing.T) {
	_, err := decryptData([]byte("legacy ciphertext"), "passphrase", "")
	if err == nil || !strings.Contains(err.Error(), "invalid aes-gcm-v2 payload") {
		t.Fatalf("expected decryptData to reject non-v2 payload, got %v", err)
	}
}

func TestEncryptAESGCMV2EmbedsArgonParameters(t *testing.T) {
	ciphertext, err := encryptData([]byte("continuum"), "passphrase", AlgoAES_GCM_V2)
	if err != nil {
		t.Fatalf("encryptData: %v", err)
	}
	if !bytes.HasPrefix(ciphertext, []byte(v2Magic)) {
		t.Fatalf("expected v2 magic prefix")
	}

	reader := bytes.NewReader(ciphertext[len(v2Magic):])
	var timeCost uint32
	var memoryCost uint32
	if err := binary.Read(reader, binary.BigEndian, &timeCost); err != nil {
		t.Fatalf("read timeCost: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &memoryCost); err != nil {
		t.Fatalf("read memoryCost: %v", err)
	}
	if timeCost != v2ArgonTime {
		t.Fatalf("timeCost = %d, want %d", timeCost, v2ArgonTime)
	}
	if memoryCost != v2ArgonMemory {
		t.Fatalf("memoryCost = %d, want %d", memoryCost, v2ArgonMemory)
	}
}
