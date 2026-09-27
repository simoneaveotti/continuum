package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// AcquireStorageLock serializes short-lived CLI operations sharing one storage.
// The advisory lock is released automatically by the OS if the process exits.
func AcquireStorageLock() (func(), error) {
	lockDir := filepath.Join(ContinuumPath(), "local")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		return nil, fmt.Errorf("cannot prepare storage lock: %w", err)
	}
	lock := flock.New(filepath.Join(lockDir, "ctx.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("cannot acquire storage lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("another Continuum operation is already using this storage; try again shortly")
	}
	return func() { _ = lock.Unlock() }, nil
}
