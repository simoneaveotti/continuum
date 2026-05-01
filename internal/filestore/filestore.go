package filestore

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CaptureType string

const (
	StateCapture    CaptureType = "state"
	ProposalCapture CaptureType = "proposal"
	RequestCapture  CaptureType = "request"
	ResponseCapture CaptureType = "response"
	DecisionCapture CaptureType = "decision"
)

func ValidateCaptureType(value string) (CaptureType, error) {
	switch CaptureType(value) {
	case StateCapture, ProposalCapture, RequestCapture, ResponseCapture, DecisionCapture:
		return CaptureType(value), nil
	default:
		return "", fmt.Errorf("invalid capture type: %q (expected state, proposal, request, response, or decision)", value)
	}
}

func capturePrefix(captureType CaptureType) string {
	switch captureType {
	case StateCapture:
		return "snapshot."
	case ProposalCapture:
		return "proposal."
	case RequestCapture:
		return "request."
	case ResponseCapture:
		return "response."
	case DecisionCapture:
		return "decision."
	default:
		return "snapshot."
	}
}

func NewCaptureName(captureType CaptureType) string {
	return fmt.Sprintf("%s%s.%s.md", capturePrefix(captureType), timestamp(), randHex6())
}

// NewSnapshotName returns a unique snapshot filename.
// Format: snapshot.20060102T150405Z.a3f2c1.md
func NewSnapshotName() string {
	return NewCaptureName(StateCapture)
}

// NewHandoffName returns a unique handoff filename.
// Format: handoff.20060102T150405Z.a3f2c1.md
func NewHandoffName() string {
	return fmt.Sprintf("handoff.%s.%s.md", timestamp(), randHex6())
}

// AtomicWrite writes content to path atomically via a temp file + rename.
// Readers never observe a partial write.
func AtomicWrite(path string, content []byte) error {
	dir := filepath.Dir(path)
	tmp := filepath.Join(dir, ".tmp."+randHex6())
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return fmt.Errorf("atomic write (stage): %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("atomic write (commit): %w", err)
	}
	return nil
}

// LatestSnapshot returns the path and filename of the most recent snapshot
// in taskDir. Returns ("", "", nil) if no snapshot exists.
func LatestSnapshot(taskDir string) (path, name string, err error) {
	return LatestCaptureOfType(taskDir, StateCapture)
}

func LatestCaptureOfType(taskDir string, captureType CaptureType) (path, name string, err error) {
	names, err := listFiles(taskDir, capturePrefix(captureType))
	if err != nil || len(names) == 0 {
		return "", "", err
	}
	name = names[len(names)-1]
	return filepath.Join(taskDir, name), name, nil
}

// LatestHandoff returns the path and filename of the most recent handoff
// in taskDir. Returns ("", "", nil) if no handoff exists.
func LatestHandoff(taskDir string) (path, name string, err error) {
	names, err := listFiles(taskDir, "handoff.")
	if err != nil || len(names) == 0 {
		return "", "", err
	}
	name = names[len(names)-1]
	return filepath.Join(taskDir, name), name, nil
}

// AllSnapshots returns all snapshot paths in taskDir, sorted chronologically.
func AllSnapshots(taskDir string) ([]string, error) {
	return AllCapturesOfType(taskDir, StateCapture)
}

// AllHandoffs returns all handoff paths in taskDir, sorted chronologically.
func AllHandoffs(taskDir string) ([]string, error) {
	return allPaths(taskDir, "handoff.")
}

func AllCapturesOfType(taskDir string, captureType CaptureType) ([]string, error) {
	return allPaths(taskDir, capturePrefix(captureType))
}

func ResolveArtifact(taskDir, name string) error {
	resolvedDir := filepath.Join(taskDir, "resolved")
	if err := os.MkdirAll(resolvedDir, 0o755); err != nil {
		return fmt.Errorf("cannot create resolved dir: %w", err)
	}
	dst := filepath.Join(resolvedDir, name)
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("resolved artifact already exists: %s", name)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot access resolved artifact: %w", err)
	}
	if err := os.Rename(filepath.Join(taskDir, name), dst); err != nil {
		return fmt.Errorf("cannot resolve artifact %s: %w", name, err)
	}
	return nil
}

func allPaths(dir, prefix string) ([]string, error) {
	names, err := listFiles(dir, prefix)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(names))
	for i, name := range names {
		paths[i] = filepath.Join(dir, name)
	}
	return paths, nil
}

func listFiles(dir, prefix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot read directory: %w", err)
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasPrefix(n, prefix) && strings.HasSuffix(n, ".md") {
			names = append(names, n)
		}
	}
	sort.Strings(names) // lexicographic = chronological for RFC3339-compact timestamps
	return names, nil
}

func timestamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

func randHex6() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%06x", time.Now().UnixNano()&0xFFFFFF)
	}
	return fmt.Sprintf("%x", b)
}
