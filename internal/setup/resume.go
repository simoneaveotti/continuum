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

func Resume() (ResumeResult, error) {
	base := ContinuumPath()
	result := ResumeResult{BasePath: base}

	if !repoExists(base) {
		return result, fmt.Errorf("continuum repository not initialized")
	}

	git := newVCS(base)
	clean, err := git.Fsck(base)
	if err != nil {
		return result, fmt.Errorf("cannot validate Continuum storage: %w", err)
	}
	if !clean || inProgress(base) {
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
		syncResult, err := Sync("")
		if err != nil {
			result.SyncWarning = err.Error()
		} else {
			result.Sync = &syncResult
		}
	} else {
		result.SyncWarning = "remote not configured; continuing in local-only mode"
	}

	projects, err := ListProjects()
	if err != nil {
		return result, err
	}
	result.Projects = projects
	result.UnsyncedCount = len(UnsyncedCommits())
	return result, nil
}
