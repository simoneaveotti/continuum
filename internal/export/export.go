package export

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"continuum/internal/events"
	"continuum/internal/filestore"
	"continuum/internal/setup"

	"golang.org/x/crypto/argon2"
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
	v2ArgonMemory  = uint32(32 * 1024) // KiB
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

func encryptData(data []byte, passphrase string, algo EncryptionAlgo) ([]byte, error) {
	algo = algo.Default()

	switch algo {
	case AlgoAES_GCM_V2, "":
		return encryptAESGCMV2(data, passphrase)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algo)
	}
}

func decryptData(data []byte, passphrase string, algo EncryptionAlgo) ([]byte, error) {
	switch algo {
	case "", AlgoAES_GCM_V2:
		return decryptAESGCMV2(data, passphrase)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algo)
	}
}

func encryptAESGCMV2(data []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, v2SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key := deriveKeyArgon2(passphrase, salt, v2ArgonTime, v2ArgonMemory, v2ArgonThreads)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)
	buf := bytes.NewBuffer(make([]byte, 0, len(v2Magic)+13+len(salt)+len(nonce)+len(ciphertext)))
	buf.WriteString(v2Magic)
	_ = binary.Write(buf, binary.BigEndian, v2ArgonTime)
	_ = binary.Write(buf, binary.BigEndian, v2ArgonMemory)
	buf.WriteByte(v2ArgonThreads)
	buf.WriteByte(byte(len(salt)))
	buf.WriteByte(byte(len(nonce)))
	buf.Write(salt)
	buf.Write(nonce)
	buf.Write(ciphertext)
	return buf.Bytes(), nil
}

func decryptAESGCMV2(data []byte, passphrase string) ([]byte, error) {
	if !strings.HasPrefix(string(data), v2Magic) {
		return nil, fmt.Errorf("invalid %s payload", AlgoAES_GCM_V2)
	}
	reader := bytes.NewReader(data[len(v2Magic):])

	var timeCost uint32
	var memoryCost uint32
	var threads uint8
	var saltLen uint8
	var nonceLen uint8

	if err := binary.Read(reader, binary.BigEndian, &timeCost); err != nil {
		return nil, fmt.Errorf("invalid v2 header: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &memoryCost); err != nil {
		return nil, fmt.Errorf("invalid v2 header: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &threads); err != nil {
		return nil, fmt.Errorf("invalid v2 header: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &saltLen); err != nil {
		return nil, fmt.Errorf("invalid v2 header: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &nonceLen); err != nil {
		return nil, fmt.Errorf("invalid v2 header: %w", err)
	}
	if saltLen == 0 || nonceLen == 0 {
		return nil, fmt.Errorf("invalid v2 header lengths")
	}

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(reader, salt); err != nil {
		return nil, fmt.Errorf("invalid v2 salt: %w", err)
	}
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(reader, nonce); err != nil {
		return nil, fmt.Errorf("invalid v2 nonce: %w", err)
	}
	ciphertext, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	key := deriveKeyArgon2(passphrase, salt, timeCost, memoryCost, threads)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func deriveKeyArgon2(passphrase string, salt []byte, timeCost, memoryCost uint32, threads uint8) []byte {
	return argon2.IDKey([]byte(passphrase), salt, timeCost, memoryCost, threads, v2KeySize)
}

func promptPassphrase() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter passphrase: ")
	pass, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return pass[:len(pass)-1], nil
}

func resolveOutputPath(customPath, task, suffix string) (string, error) {
	if customPath != "" {
		if filepath.Ext(customPath) == "" {
			if err := os.MkdirAll(customPath, 0o755); err != nil {
				return "", fmt.Errorf("cannot create output directory: %w", err)
			}
			timestamp := time.Now().Format("20060102-150405")
			return filepath.Join(customPath, fmt.Sprintf("%s-%s.%s", task, timestamp, suffix)), nil
		}
		return customPath, nil
	}
	exportsDir := filepath.Join(setup.ContinuumPath(), "exports")
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create exports directory: %w", err)
	}
	timestamp := time.Now().Format("20060102-150405")
	return filepath.Join(exportsDir, fmt.Sprintf("%s-%s.%s", task, timestamp, suffix)), nil
}

func addSingleFileToZip(zipWriter *zip.Writer, srcPath, zipName string) error {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	w, err := zipWriter.Create(zipName)
	if err != nil {
		return err
	}
	if _, err := w.Write(content); err != nil {
		return err
	}
	return nil
}

func addPathToZip(zipWriter *zip.Writer, root, relPath string) error {
	fullPath := filepath.Join(root, filepath.FromSlash(relPath))
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			child := filepath.ToSlash(filepath.Join(relPath, entry.Name()))
			if err := addPathToZip(zipWriter, root, child); err != nil {
				return err
			}
		}
		return nil
	}
	return addSingleFileToZip(zipWriter, fullPath, relPath)
}

func addManifest(zipWriter *zip.Writer, manifest ArchiveManifest) error {
	w, err := zipWriter.Create(manifestName)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

// gatherTaskFileNames returns the relative filenames of all snapshot, handoff,
// and notes files in taskDir.
func gatherTaskFileNames(taskDir string) []string {
	var names []string
	if paths, err := filestore.AllSnapshots(taskDir); err == nil {
		for _, p := range paths {
			names = append(names, filepath.Base(p))
		}
	}
	if paths, err := filestore.AllHandoffs(taskDir); err == nil {
		for _, p := range paths {
			names = append(names, filepath.Base(p))
		}
	}
	names = append(names, "notes.md")
	return names
}

func writeZipArchive(relPaths []string, outputPath, archiveBase string, manifest ArchiveManifest) (string, error) {
	output, err := resolveOutputPath(outputPath, archiveBase, "zip")
	if err != nil {
		return "", err
	}

	zipFile, err := os.Create(output)
	if err != nil {
		return "", fmt.Errorf("cannot create zip: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	if err := addManifest(zipWriter, manifest); err != nil {
		return "", fmt.Errorf("cannot write archive manifest: %w", err)
	}
	root := setup.ContinuumPath()
	for _, relPath := range relPaths {
		if err := addPathToZip(zipWriter, root, relPath); err != nil {
			return "", fmt.Errorf("cannot add %s to archive: %w", relPath, err)
		}
	}
	return output, nil
}

func writeEncryptedZipArchive(relPaths []string, outputPath, archiveBase string, algo EncryptionAlgo, manifest ArchiveManifest) (string, error) {
	algo = algo.Default()

	passphrase, err := promptPassphrase()
	if err != nil {
		return "", fmt.Errorf("passphrase error: %w", err)
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	if err := addManifest(zipWriter, manifest); err != nil {
		return "", fmt.Errorf("cannot write archive manifest: %w", err)
	}
	root := setup.ContinuumPath()
	for _, relPath := range relPaths {
		if err := addPathToZip(zipWriter, root, relPath); err != nil {
			return "", fmt.Errorf("cannot add %s to archive: %w", relPath, err)
		}
	}
	zipWriter.Close()

	encrypted, err := encryptData(buf.Bytes(), passphrase, algo)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}

	output, err := resolveOutputPath(outputPath, archiveBase, "zip.enc")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(output, encrypted, 0o600); err != nil {
		return "", fmt.Errorf("cannot write encrypted file: %w", err)
	}

	return output, nil
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

func promptDecryptPassphrase() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter decryption passphrase: ")
	pass, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return pass[:len(pass)-1], nil
}

func ImportArchive(zipPath string, decrypt bool, algo EncryptionAlgo) (string, error) {
	if zipPath == "" {
		return "", fmt.Errorf("zip path is required")
	}

	fileData, err := os.ReadFile(zipPath)
	if err != nil {
		return "", fmt.Errorf("cannot read file: %w", err)
	}

	if decrypt {
		passphrase, err := promptDecryptPassphrase()
		if err != nil {
			return "", fmt.Errorf("passphrase error: %w", err)
		}
		fileData, err = decryptData(fileData, passphrase, algo)
		if err != nil {
			return "", fmt.Errorf("decryption failed: %w", err)
		}
	}

	reader := bytes.NewReader(fileData)
	zipReader, err := zip.NewReader(reader, int64(len(fileData)))
	if err != nil {
		return "", fmt.Errorf("invalid zip: %w", err)
	}

	manifest := readArchiveManifest(zipReader)
	base := setup.ContinuumPath()
	for _, f := range zipReader.File {
		if f.Name == manifestName || f.Name == "IMPORT_INSTRUCTIONS.md" {
			continue
		}
		destDir := base
		if manifest == nil && !strings.Contains(f.Name, "/") {
			taskName, projectName := extractTaskInfo(zipPath, zipReader)
			if taskName == "" || projectName == "" {
				return "", fmt.Errorf("could not determine task import target from zip")
			}
			destDir = filepath.Join(base, "projects", projectName, "tasks", taskName)
			if f.Name == "project.md" {
				destDir = filepath.Join(base, "projects", projectName)
			}
		}
		if err := extractZipFile(f, destDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", f.Name, err)
		}
	}

	if manifest != nil {
		switch manifest.Kind {
		case ArchiveTask:
			if manifest.Task != "" {
				_ = events.Append(manifestProjectForEvent(manifest), manifest.Task, "import", "ok", "task archive imported")
				return manifest.Task, nil
			}
			_ = events.Append(manifestProjectForEvent(manifest), "", "import", "ok", "task archive imported")
			return "task archive", nil
		case ArchiveProject:
			_ = events.Append(manifestProjectForEvent(manifest), "", "import", "ok", importProjectDetail(manifest.Projects))
			if len(manifest.Projects) == 1 {
				return "project " + manifest.Projects[0], nil
			}
			return fmt.Sprintf("%d projects", len(manifest.Projects)), nil
		case ArchiveSession:
			_ = events.Append("", "", "import", "ok", "session archive imported")
			return "session", nil
		}
	}
	taskName, _ := extractTaskInfo(zipPath, zipReader)
	if taskName == "" {
		_ = events.Append("", "", "import", "ok", "archive imported")
		return "archive", nil
	}
	_ = events.Append("", taskName, "import", "ok", "task archive imported")
	return taskName, nil
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

func readArchiveManifest(zipReader *zip.Reader) *ArchiveManifest {
	for _, f := range zipReader.File {
		if f.Name != manifestName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil
		}
		var manifest ArchiveManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil
		}
		return &manifest
	}
	return nil
}

// extractTaskInfo derives the task name and project name from the snapshot
// inside the zip. Falls back to the zip filename for the task name if needed.
// Project name has no fallback — it must be present in the snapshot.
func extractTaskInfo(zipPath string, zipReader *zip.Reader) (taskName, projectName string) {
	for _, f := range zipReader.File {
		if !strings.HasPrefix(f.Name, "snapshot.") || !strings.HasSuffix(f.Name, ".md") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			break
		}
		content, _ := io.ReadAll(rc)
		rc.Close()
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "## Task" && i+1 < len(lines) {
				taskName = strings.TrimSpace(lines[i+1])
			}
			if trimmed == "## Project" && i+1 < len(lines) {
				projectName = strings.TrimSpace(lines[i+1])
			}
		}
		break
	}

	if taskName == "" {
		// fallback: derive from zip filename
		base := filepath.Base(zipPath)
		name := base
		for ext := filepath.Ext(name); ext != ""; ext = filepath.Ext(name) {
			name = strings.TrimSuffix(name, ext)
		}
		name = strings.ReplaceAll(name, "-share", "")
		name = strings.ReplaceAll(name, "-encrypted", "")
		taskName = strings.TrimSpace(name)
	}

	return taskName, projectName
}

// extractTaskName is kept for backward compatibility with existing tests.
func extractTaskName(zipPath string, zipReader *zip.Reader) string {
	name, _ := extractTaskInfo(zipPath, zipReader)
	return name
}

// extractZipFile safely extracts a single zip entry into destDir,
// preventing zip slip by validating the resolved path.
func extractZipFile(f *zip.File, destDir string) error {
	cleanName := filepath.Clean(f.Name)
	if filepath.IsAbs(cleanName) {
		return fmt.Errorf("invalid path in zip: %s", f.Name)
	}
	destPath := filepath.Join(destDir, cleanName)
	if !strings.HasPrefix(destPath+string(os.PathSeparator), destDir+string(os.PathSeparator)) {
		return fmt.Errorf("invalid path in zip: %s", f.Name)
	}

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("cannot open zip entry: %w", err)
	}
	defer rc.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("cannot create parent directory: %w", err)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("cannot write file: %w", err)
	}

	return nil
}
