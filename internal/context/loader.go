package context

import (
	"fmt"
	"os"
	"path/filepath"
)

type ContextData struct {
	Profile  string
	Project  string
	Snapshot string
	Handoff  string
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("missing file: %s", path)
	}
	return string(data), nil
}

func Load(task string) (*ContextData, error) {
	base := ".continuum"

	profile, err := readFile(filepath.Join(base, "profile.md"))
	if err != nil {
		return nil, err
	}

	project, err := readFile(filepath.Join(base, "project.md"))
	if err != nil {
		return nil, err
	}

	snapshotPath := filepath.Join(base, "tasks", task, "snapshot.md")
	snapshot, err := readFile(snapshotPath)
	if err != nil {
		return nil, err
	}

	handoffPath := filepath.Join(base, "tasks", task, "handoff.md")
	handoff, err := readFile(handoffPath)
	if err != nil {
		return nil, err
	}

	return &ContextData{
		Profile:  profile,
		Project:  project,
		Snapshot: snapshot,
		Handoff:  handoff,
	}, nil
}
