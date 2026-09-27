package events

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"continuum/internal/identity"
	"continuum/internal/storagepath"
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

// tailChunk is how much of the activity log is pulled in per backwards step.
const tailChunk = 32 * 1024

// ReadTail returns the newest limit events without parsing the whole log.
//
// The activity log is append-only JSONL and a real one reaches 1.4 MB, so
// reading it from the start to display the last couple of hundred events wastes
// almost all of that work. This walks the file backwards a chunk at a time and
// stops as soon as it has enough complete lines.
//
// The returned offset is the size of the file, so a caller can keep reading
// incrementally from the end afterwards. A trailing line without a newline is
// an unflushed write and is skipped rather than parsed as a partial record.
func ReadTail(limit int) ([]Event, int64, error) {
	if limit <= 0 {
		return nil, 0, nil
	}
	path := ActivityPath()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("cannot open activity stream: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("cannot stat activity stream: %w", err)
	}
	size := info.Size()
	if size == 0 {
		return nil, 0, nil
	}

	var lines []string
	end := size
	// Stop as soon as we hold enough lines plus the one straddling the chunk
	// boundary, which we may only have read in part.
	for end > 0 && len(lines) <= limit {
		start := max(int64(0), end-tailChunk)
		buf := make([]byte, end-start)
		if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
			return nil, 0, fmt.Errorf("cannot read activity stream: %w", err)
		}
		chunk := string(buf)
		// Only the last record can be incomplete. Drop it on the first pass so a
		// writer that has not flushed its newline cannot leak a partial event.
		if end == size && !strings.HasSuffix(chunk, "\n") {
			if idx := strings.LastIndexByte(chunk, '\n'); idx >= 0 {
				chunk = chunk[:idx+1]
			} else {
				chunk = ""
			}
		}
		// Only whole lines are usable: the first one may continue from the
		// previous chunk, so drop it unless we reached the start of the file.
		if start > 0 {
			if idx := strings.IndexByte(chunk, '\n'); idx >= 0 {
				chunk = chunk[idx+1:]
			} else {
				chunk = ""
			}
		}
		if chunk != "" {
			lines = append(strings.Split(strings.TrimSuffix(chunk, "\n"), "\n"), lines...)
		}
		end = start
	}

	// Reverse into chronological order, keeping only the newest limit.
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	items := make([]Event, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var item Event
		if uerr := json.Unmarshal([]byte(line), &item); uerr == nil {
			items = append(items, item)
		}
	}
	return items, size, nil
}

// ErrLogRewritten reports that a byte offset no longer refers to the same
// activity log the caller was reading. ctx repair --activity rewrites the file
// in place, which can shrink it or move every byte; a long lived watch holding
// an old offset would otherwise seek into the middle of a record, and every
// event from there on would fail to parse and vanish without a word.
var ErrLogRewritten = errors.New("activity log was rewritten")

// checkOffsetIsLineStart verifies that offset still points at the beginning of
// a record: it must be inside the file and the byte before it must be a newline.
func checkOffsetIsLineStart(f *os.File, offset int64) error {
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat activity stream: %w", err)
	}
	if offset > info.Size() {
		return ErrLogRewritten
	}
	if offset == 0 {
		return nil
	}
	var prev [1]byte
	if _, err := f.ReadAt(prev[:], offset-1); err != nil {
		return ErrLogRewritten
	}
	if prev[0] != '\n' {
		return ErrLogRewritten
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
		if err := checkOffsetIsLineStart(f, offset); err != nil {
			return nil, offset, err
		}
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
	return filepath.Join(storagepath.ContinuumPath(), activityRelPath)
}

func ActivityRelPath() string {
	return activityRelPath
}
