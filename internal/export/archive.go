package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"continuum/internal/events"
	"continuum/internal/filestore"
	"continuum/internal/setup"
)

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
