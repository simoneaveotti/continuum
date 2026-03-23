package task

import (
	"fmt"
	"os"
	"path/filepath"
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

func snapshotTemplate(task string) string {
	return fmt.Sprintf(`# TASK SNAPSHOT

## Task
%s

## Objective
...

## Current State
- ...

## Decisions (Locked)
- ...

## Relevant Files
- ...

## Last Step Completed
- ...

## Next Step
- ...

## Active Issues
- ...

## Constraints
- ...

## Git Context
- branch: unknown
- status: unknown

## Things Not To Revisit
- ...

## Suggested Prompt Seed
Continue this task using the snapshot, handoff, and project context. Do not reopen locked decisions unless new evidence appears.

## Last Updated
...
`, task)
}

func handoffTemplate(task string) string {
	return fmt.Sprintf(`# TASK HANDOFF

## Task
%s

## Objective
...

## What Was Done
- ...

## Current State
- ...

## Decisions Confirmed
- ...

## Relevant Files
- ...

## Risks / Caveats
- ...

## Next Recommended Step
- ...

## Agent Notes
- ...

# SURVEY FOR NEXT AGENT

## Unresolved Questions
- ...

## Assumptions To Validate
- ...

## Things That Might Be Wrong
- ...

## Missing Context
- ...

## Ask Before Proceeding If
- ...

## Suggested First Action
- ...

## Last Updated
...
`, task)
}

const notesTemplate = `# TASK NOTES

- ...
`

func Start(task string) error {
	if task == "" {
		return fmt.Errorf("task name is required")
	}

	taskDir := filepath.Join(".continuum", "tasks", task)

	if err := ensureDir(taskDir); err != nil {
		return fmt.Errorf("cannot create task directory %s: %w", taskDir, err)
	}

	if err := ensureFile(filepath.Join(taskDir, "snapshot.md"), snapshotTemplate(task)); err != nil {
		return fmt.Errorf("cannot create snapshot.md: %w", err)
	}

	if err := ensureFile(filepath.Join(taskDir, "handoff.md"), handoffTemplate(task)); err != nil {
		return fmt.Errorf("cannot create handoff.md: %w", err)
	}

	if err := ensureFile(filepath.Join(taskDir, "notes.md"), notesTemplate); err != nil {
		return fmt.Errorf("cannot create notes.md: %w", err)
	}

	fmt.Printf("Task '%s' initialized in %s\n", task, taskDir)
	return nil
}
