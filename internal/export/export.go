package export

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"continuum/internal/context"
)

func WriteTaskExport(task string, content string) (string, error) {
	if task == "" {
		return "", fmt.Errorf("task name is required")
	}

	exportsDir := filepath.Join(".continuum", "exports")
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create exports directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.md", task, timestamp)
	outputPath := filepath.Join(exportsDir, filename)

	header := fmt.Sprintf(`# CONTINUUM EXPORT

## Task
%s

## Generated At
%s

`, task, time.Now().Format("2006-01-02 15:04:05 MST"))

	finalContent := header + content + "\n"

	if err := os.WriteFile(outputPath, []byte(finalContent), 0o644); err != nil {
		return "", fmt.Errorf("cannot write export file: %w", err)
	}

	return outputPath, nil
}

func LoadAndExport(task string) (string, error) {
	ctxData, err := context.Load(task)
	if err != nil {
		return "", err
	}

	content := context.BuildPromptOnlyPackage(ctxData)
	return WriteTaskExport(task, content)
}
