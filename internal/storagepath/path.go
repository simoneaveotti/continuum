package storagepath

import (
	"os"
	"path/filepath"
)

// ContinuumPath returns the base directory for Continuum storage.
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
		return filepath.Join(cwd, ".ctx")
	}
	return filepath.Join(home, ".ctx")
}
