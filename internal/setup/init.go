package setup

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	profileTemplate = `# PROFILE

## Working Preferences
- concise operational summaries
- avoid re-explaining known context
- preserve decisions

## Rules
- do not store secrets in shared files
- distinguish facts vs assumptions

## Output Style
- high-density
- actionable
- copy-paste friendly
`

	projectTemplate = `# PROJECT

## Name
...

## Summary
...

## Stack
...

## Constraints
...

## Important Files
...

## Notes
...
`
)

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

func Init() error {
	base := ".continuum"

	dirs := []string{
		base,
		filepath.Join(base, "tasks"),
		filepath.Join(base, "local"),
		filepath.Join(base, "exports"),
	}

	for _, dir := range dirs {
		if err := ensureDir(dir); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", dir, err)
		}
	}

	if err := ensureFile(filepath.Join(base, "profile.md"), profileTemplate); err != nil {
		return fmt.Errorf("cannot create profile.md: %w", err)
	}

	if err := ensureFile(filepath.Join(base, "project.md"), projectTemplate); err != nil {
		return fmt.Errorf("cannot create project.md: %w", err)
	}

	fmt.Println("Continuum initialized in .continuum/")
	return nil
}
