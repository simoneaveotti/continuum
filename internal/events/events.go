package events

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"continuum/internal/identity"
)

const activityRelPath = "events/activity.ndjson"

type Event struct {
	Timestamp string `json:"timestamp"`
	Agent     string `json:"agent"`
	Host      string `json:"host"`
	Project   string `json:"project,omitempty"`
	Task      string `json:"task,omitempty"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	File      string `json:"file,omitempty"`
}

func Append(project, task, eventType, status, detail string) error {
	return AppendWithFile(project, task, eventType, status, detail, "")
}

func AppendWithFile(project, task, eventType, status, detail, file string) error {
	path := ActivityPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cannot create events directory: %w", err)
	}

	payload := Event{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Agent:     identity.AgentName(),
		Host:      identity.HostName(),
		Project:   project,
		Task:      task,
		Type:      eventType,
		Status:    status,
		Detail:    detail,
		File:      file,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("cannot marshal event: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("cannot open activity stream: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("cannot append event: %w", err)
	}
	return nil
}

func ReadFromOffset(offset int64) ([]Event, int64, error) {
	path := ActivityPath()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, offset, fmt.Errorf("cannot open activity stream: %w", err)
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return nil, offset, fmt.Errorf("cannot seek activity stream: %w", err)
		}
	}

	reader := bufio.NewReader(f)
	var items []Event
	current := offset

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			current += int64(len(line))
			line = strings.TrimSpace(line)
			if line != "" {
				var item Event
				if uerr := json.Unmarshal([]byte(line), &item); uerr == nil {
					items = append(items, item)
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return items, current, fmt.Errorf("cannot read activity stream: %w", err)
		}
	}

	return items, current, nil
}

func ActivityPath() string {
	return filepath.Join(continuumPath(), activityRelPath)
}

func ActivityRelPath() string {
	return activityRelPath
}

func continuumPath() string {
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
		return filepath.Join(cwd, ".continuum")
	}
	return filepath.Join(home, ".continuum")
}
