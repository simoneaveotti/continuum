package setup

import "fmt"

type ResumeResult struct {
	BasePath      string
	RepairMessage string
	Sync          *SyncResult
	SyncWarning   string
	Projects      []string
	UnsyncedCount int
}

// Resume prepares Continuum storage for a new work session. onProgress, if
// non-nil, is called with a short message before each phase starts so the
// caller can print live progress instead of only a final summary.
func Resume(onProgress func(string)) (ResumeResult, error) {
	progress := func(msg string) {
		if onProgress != nil {
			onProgress(msg)
		}
	}

	base := ContinuumPath()
	result := ResumeResult{BasePath: base}

	if !repoExists(base) {
		return result, fmt.Errorf("continuum repository not initialized")
	}

	progress("Checking storage integrity...")
	git := newVCS(base)
	clean, err := git.Fsck(base)
	if err != nil {
		return result, fmt.Errorf("cannot validate Continuum storage: %w", err)
	}
	if !clean || inProgress(base) {
		progress("Repairing storage...")
		msg, err := Repair()
		if err != nil {
			return result, err
		}
		result.RepairMessage = msg
		if !repoExists(base) {
			return result, fmt.Errorf("continuum repository not initialized")
		}
		if dirty, err := dirtyWorktreeSummary(base); err == nil && dirty != "" {
			return result, fmt.Errorf("Continuum storage needs manual recovery. %s", msg)
		}
		if msg != "Intermediate git state cleared. Repository recovered." && msg != "No issues detected." {
			return result, fmt.Errorf("Continuum storage needs manual recovery. %s", msg)
		}
	}

	if dirty, err := dirtyWorktreeSummary(base); err != nil {
		return result, fmt.Errorf("cannot inspect local Continuum changes: %w", err)
	} else if dirty != "" {
		return result, fmt.Errorf("Continuum storage has local changes. Commit, stash, or discard them in %s before running 'ctx resume' again.", base)
	}

	if git.HasRemote(base, "origin") {
		progress("Syncing with remote...")
		syncResult, err := Sync("")
		if err != nil {
			result.SyncWarning = err.Error()
		} else {
			result.Sync = &syncResult
		}
	} else {
		result.SyncWarning = "remote not configured; continuing in local-only mode"
	}

	progress("Collecting projects...")
	projects, err := ListProjects()
	if err != nil {
		return result, err
	}
	result.Projects = projects
	result.UnsyncedCount = len(UnsyncedCommits())
	return result, nil
}
