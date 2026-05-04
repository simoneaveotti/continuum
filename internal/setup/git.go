package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"continuum/internal/events"
	"continuum/internal/vcs"
)

const gitIgnoreContent = `# File temporanei di cattura
*.tmp.*

# Export e share non vanno in git
exports/

# Local state (unsynced marker, log)
local/
*.log

# OS
.DS_Store
Thumbs.db
._*
`

const gitAttributesContent = `events/activity.ndjson merge=union
`

func newVCS(base string) vcs.VCS {
	return vcs.NewGit(filepath.Join(base, "local", "git.log"))
}

const unsyncedRelPath = "local/unsynced"

func repoExists(base string) bool {
	info, err := os.Stat(filepath.Join(base, ".git"))
	return err == nil && info.IsDir()
}

func ensureGitRepo(base string, force bool) error {
	git := newVCS(base)
	hadRepo := repoExists(base)

	if err := writeFile(filepath.Join(base, ".gitignore"), []byte(gitIgnoreContent), force); err != nil {
		return fmt.Errorf("cannot create .gitignore: %w", err)
	}
	if err := writeFile(filepath.Join(base, ".gitattributes"), []byte(gitAttributesContent), force); err != nil {
		return fmt.Errorf("cannot create .gitattributes: %w", err)
	}

	if err := git.Init(base); err != nil {
		return fmt.Errorf("cannot initialize git repository: %w", err)
	}

	if remote := strings.TrimSpace(os.Getenv("CONTINUUM_REMOTE")); remote != "" {
		if err := git.AddRemote(base, "origin", remote); err != nil {
			return fmt.Errorf("cannot configure git remote: %w", err)
		}
	}

	if hadRepo {
		return nil
	}

	files, err := listTrackedFiles(base)
	if err != nil {
		return fmt.Errorf("cannot list git seed files: %w", err)
	}
	if err := git.Commit(base, "init: continuum initialized", files); err != nil {
		return fmt.Errorf("cannot create initial git commit: %w", err)
	}
	return nil
}

func CommitFiles(message string, files []string) error {
	base := ContinuumPath()
	if !repoExists(base) {
		return nil
	}
	if err := newVCS(base).Commit(base, message, files); err != nil {
		return err
	}
	return nil
}

func PushBestEffort() {
	base := ContinuumPath()
	if !repoExists(base) {
		return
	}
	git := newVCS(base)
	if !git.HasRemote(base, "origin") {
		return
	}
	if err := git.Push(base); err != nil {
		if head, herr := git.HeadCommit(base); herr == nil && head != "" {
			_ = markUnsynced(head)
		}
		return
	}
	_ = ClearUnsynced()
}

func PullLatest() error {
	base := ContinuumPath()
	if !repoExists(base) {
		return nil
	}
	git := newVCS(base)
	if !git.HasRemote(base, "origin") {
		return nil
	}
	if dirty, err := dirtyWorktreeSummary(base); err == nil && dirty != "" {
		return fmt.Errorf("Local Continuum storage has uncommitted changes. Showing local state.")
	}
	if err := git.Pull(base); err != nil {
		return fmt.Errorf("Could not pull from origin. Showing local state.")
	}
	return nil
}

func listTrackedFiles(base string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	cmd.Dir = base
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var files []string
	for _, entry := range bytes.Split(out, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		files = append(files, filepath.ToSlash(string(entry)))
	}
	sort.Strings(files)
	return files, nil
}

func StagePaths(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	base := ContinuumPath()
	args := append([]string{"add", "--"}, paths...)
	if err := newVCS(base).Execute(base, args...); err != nil {
		return fmt.Errorf("cannot stage files: %w", err)
	}
	return nil
}

func StageDeletedPaths(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	base := ContinuumPath()
	git := newVCS(base)
	for _, p := range paths {
		if err := git.Execute(base, "add", "-A", "--", p); err != nil {
			return fmt.Errorf("cannot stage deletions for %s: %w", p, err)
		}
	}
	return nil
}

type SyncResult struct {
	PullCount         int
	PushCount         int
	RemoteAdded       bool
	Bootstrapped      bool
	LogEntry          string
	Preference        string
	RemoteAheadBefore int
	LocalAheadBefore  int
}

type SyncOptions struct {
	Remote string
	Prefer string
	Force  bool
}

func Sync(remote string) (SyncResult, error) {
	return SyncWithOptions(SyncOptions{Remote: remote})
}

func SyncWithOptions(options SyncOptions) (SyncResult, error) {
	base := ContinuumPath()
	if !repoExists(base) {
		return SyncResult{}, fmt.Errorf("continuum repository not initialized")
	}
	git := newVCS(base)
	beforeLog := LastLogEntry()

	result := SyncResult{}
	result.Preference = options.Prefer
	if options.Force && options.Prefer == "" {
		return result, fmt.Errorf("sync --force requires --prefer=local or --prefer=remote")
	}
	if options.Prefer != "" && options.Prefer != "local" && options.Prefer != "remote" {
		return result, fmt.Errorf("invalid sync preference %q", options.Prefer)
	}
	if options.Prefer != "" && !options.Force {
		return result, fmt.Errorf("sync preference %q requires confirmation or --force", options.Prefer)
	}

	if options.Remote != "" && !git.HasRemote(base, "origin") {
		if err := git.AddRemote(base, "origin", options.Remote); err != nil {
			return result, fmt.Errorf("cannot add remote: %w", err)
		}
		result.RemoteAdded = true
	}

	if !git.HasRemote(base, "origin") {
		return result, fmt.Errorf("remote not configured — use ctx sync --remote=<url>")
	}
	remoteMainExists := hasRemoteMain(git, base)
	if options.Prefer == "local" {
		if dirty, err := dirtyWorktreeSummary(base); err == nil && dirty != "" {
			if err := preserveLocalChanges(base); err != nil {
				return result, err
			}
		}
		if remoteMainExists {
			if err := git.Execute(base, "fetch", "origin", "main", "--quiet"); err != nil {
				_ = events.Append("", "", "sync", "error", "fetch failed")
				result.LogEntry = newLogEntrySince(beforeLog)
				return result, fmt.Errorf("Could not fetch from origin.")
			}
			result.RemoteAheadBefore = countRange(git, base, "HEAD..origin/main")
			result.LocalAheadBefore = countRange(git, base, "origin/main..HEAD")
			result.PullCount = result.RemoteAheadBefore
			result.PushCount = result.LocalAheadBefore
		} else {
			result.Bootstrapped = true
			result.PushCount = countRange(git, base, "HEAD")
		}
		if result.PushCount > 0 || result.PullCount > 0 || result.Bootstrapped {
			if err := forcePushLocalToRemote(base); err != nil {
				if head, herr := git.HeadCommit(base); herr == nil && head != "" {
					_ = markUnsynced(head)
				}
				_ = events.Append("", "", "sync", "error", "push failed")
				result.LogEntry = newLogEntrySince(beforeLog)
				return result, fmt.Errorf("Could not push to origin.")
			}
		}
		_ = ClearUnsynced()
		if err := events.Append("", "", "sync", "ok", fmt.Sprintf("pull=%d push=%d", result.PullCount, result.PushCount)); err == nil {
			_ = StagePaths([]string{events.ActivityRelPath()})
			_ = CommitFiles("sync: activity stream updated", []string{events.ActivityRelPath()})
		}
		result.LogEntry = newLogEntrySince(beforeLog)
		return result, nil
	}
	if options.Prefer == "remote" {
		if remoteMainExists {
			if err := git.Execute(base, "fetch", "origin", "main", "--quiet"); err != nil {
				_ = events.Append("", "", "sync", "error", "fetch failed")
				result.LogEntry = newLogEntrySince(beforeLog)
				return result, fmt.Errorf("Could not fetch from origin.")
			}
			result.RemoteAheadBefore = countRange(git, base, "HEAD..origin/main")
			result.LocalAheadBefore = countRange(git, base, "origin/main..HEAD")
			result.PullCount = result.RemoteAheadBefore
		}
		if err := alignLocalToRemote(base); err != nil {
			return result, err
		}
		result.PushCount = 0
		_ = ClearUnsynced()
		if err := events.Append("", "", "sync", "ok", fmt.Sprintf("pull=%d push=%d", result.PullCount, result.PushCount)); err == nil {
			_ = StagePaths([]string{events.ActivityRelPath()})
			_ = CommitFiles("sync: activity stream updated", []string{events.ActivityRelPath()})
		}
		result.LogEntry = newLogEntrySince(beforeLog)
		return result, nil
	}
	if dirty, err := dirtyWorktreeSummary(base); err == nil && dirty != "" {
		return result, fmt.Errorf("Local Continuum storage has uncommitted changes:\n%s\nResolve with 'ctx sync --prefer=local' to preserve local changes or 'ctx sync --prefer=remote' to discard them.", dirty)
	}
	if remoteMainExists {
		if err := git.Execute(base, "fetch", "origin", "main", "--quiet"); err != nil {
			_ = events.Append("", "", "sync", "error", "fetch failed")
			result.LogEntry = newLogEntrySince(beforeLog)
			return result, fmt.Errorf("Could not fetch from origin.")
		}
		result.PullCount = countRange(git, base, "HEAD..origin/main")
		if err := git.Pull(base); err != nil {
			_ = events.Append("", "", "sync", "error", "pull failed")
			result.LogEntry = newLogEntrySince(beforeLog)
			return result, fmt.Errorf("Could not pull from origin.")
		}
		result.PushCount = countRange(git, base, "origin/main..HEAD")
	} else {
		result.Bootstrapped = true
		result.PushCount = countRange(git, base, "HEAD")
	}

	if result.PushCount > 0 {
		if err := git.Push(base); err != nil {
			if head, herr := git.HeadCommit(base); herr == nil && head != "" {
				_ = markUnsynced(head)
			}
			_ = events.Append("", "", "sync", "error", "push failed")
			result.LogEntry = newLogEntrySince(beforeLog)
			return result, fmt.Errorf("Could not push to origin.")
		}
	}

	_ = ClearUnsynced()
	if err := events.Append("", "", "sync", "ok", fmt.Sprintf("pull=%d push=%d", result.PullCount, result.PushCount)); err == nil {
		_ = StagePaths([]string{events.ActivityRelPath()})
		_ = CommitFiles("sync: activity stream updated", []string{events.ActivityRelPath()})
	}
	result.LogEntry = newLogEntrySince(beforeLog)
	return result, nil
}

func preserveLocalChanges(base string) error {
	git := newVCS(base)
	if err := git.Execute(base, "add", "-A"); err != nil {
		return fmt.Errorf("cannot stage local Continuum changes: %w", err)
	}
	if err := git.Commit(base, "sync: preserve local continuum changes", nil); err != nil {
		return fmt.Errorf("cannot commit local Continuum changes: %w", err)
	}
	return nil
}

func alignLocalToRemote(base string) error {
	git := newVCS(base)
	if hasRemoteMain(git, base) {
		if err := git.Execute(base, "fetch", "origin", "main", "--quiet"); err != nil {
			return fmt.Errorf("cannot fetch remote Continuum state: %w", err)
		}
		if err := git.Execute(base, "reset", "--hard", "origin/main"); err != nil {
			return fmt.Errorf("cannot align local Continuum state to origin/main: %w", err)
		}
	} else {
		if err := git.Execute(base, "reset", "--hard", "HEAD"); err != nil {
			return fmt.Errorf("cannot discard local Continuum changes: %w", err)
		}
	}
	if err := git.Execute(base, "clean", "-fd"); err != nil {
		return fmt.Errorf("cannot clean local Continuum files: %w", err)
	}
	return nil
}

func forcePushLocalToRemote(base string) error {
	git := newVCS(base)
	if err := git.Execute(base, "push", "--force-with-lease", "origin", "HEAD:main"); err != nil {
		return fmt.Errorf("cannot push local Continuum state to origin: %w", err)
	}
	return nil
}

func countRange(g gitRangeCounter, base, rangeSpec string) int {
	if rangeSpec == "" {
		return 0
	}
	if n, err := g.RevListCount(base, rangeSpec); err == nil {
		return n
	}
	return 0
}

func hasRemoteMain(_ vcs.VCS, base string) bool {
	cmd := exec.Command("git", "ls-remote", "--exit-code", "--heads", "origin", "main")
	cmd.Dir = base
	cmd.Env = os.Environ()
	return cmd.Run() == nil
}

func dirtyWorktreeSummary(base string) (string, error) {
	cmd := exec.Command("git", "status", "--short")
	cmd.Dir = base
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cannot inspect git status: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func newLogEntrySince(previous string) string {
	current := LastLogEntry()
	if current == "" || current == previous {
		return ""
	}
	return current
}

func Repair() (string, error) {
	base := ContinuumPath()
	if !repoExists(base) {
		return "", fmt.Errorf("continuum repository not initialized")
	}
	git := newVCS(base)

	clean, err := git.Fsck(base)
	if err != nil {
		return "", fmt.Errorf("cannot run git fsck: %w", err)
	}

	if clean {
		if inProgress(base) {
			if err := git.AbortInProgress(base); err != nil {
				return "", fmt.Errorf("cannot abort in-progress state: %w", err)
			}
			_ = events.Append("", "", "repair", "ok", "intermediate git state cleared")
			return "Intermediate git state cleared. Repository recovered.", nil
		}
		_ = events.Append("", "", "repair", "ok", "no issues detected")
		return "No issues detected.", nil
	}

	ts := backupTimestamp()
	backupPath, err := backupContinuum(base, ts)
	if err != nil {
		return "", fmt.Errorf("cannot create backup: %w", err)
	}
	msg := fmt.Sprintf("Backup created at %s", backupPath)
	if remote, rerr := git.RemoteURL(base, "origin"); rerr == nil && remote != "" {
		restored := fmt.Sprintf("%s-restored-%s", base, ts)
		msg = fmt.Sprintf("%s\nTo restore from the remote: git clone %s %s", msg, remote, restored)
	}
	_ = events.Append("", "", "repair", "ok", "backup created")
	return msg, nil
}

func RepairActivityLog() (string, error) {
	base := ContinuumPath()
	if !repoExists(base) {
		return "", fmt.Errorf("continuum repository not initialized")
	}

	path := events.ActivityPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "Activity log is clean.", nil
		}
		return "", fmt.Errorf("cannot read activity log: %w", err)
	}

	normalized, duplicatesRemoved, invalidRemoved, changed := normalizeActivityLog(data)
	if !changed {
		return "Activity log is clean.", nil
	}

	if err := writeAtomic(path, normalized, 0o644); err != nil {
		return "", fmt.Errorf("cannot rewrite activity log: %w", err)
	}
	if err := CommitFiles("repair: normalize activity log", []string{events.ActivityRelPath()}); err != nil {
		return "", err
	}
	PushBestEffort()

	switch {
	case duplicatesRemoved > 0:
		return fmt.Sprintf("Activity log repaired: %d duplicates removed.", duplicatesRemoved), nil
	case invalidRemoved > 0:
		return fmt.Sprintf("Activity log repaired: %d invalid lines removed.", invalidRemoved), nil
	default:
		return "Activity log repaired.", nil
	}
}

type activityLine struct {
	raw       string
	timestamp string
	order     int
}

func normalizeActivityLog(data []byte) ([]byte, int, int, bool) {
	rawLines := strings.Split(string(data), "\n")
	items := make([]activityLine, 0, len(rawLines))
	seen := make(map[string]struct{}, len(rawLines))
	duplicatesRemoved := 0
	invalidRemoved := 0

	for idx, line := range rawLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			invalidRemoved++
			continue
		}
		if _, ok := seen[line]; ok {
			duplicatesRemoved++
			continue
		}
		seen[line] = struct{}{}
		items = append(items, activityLine{
			raw:       line,
			timestamp: event.Timestamp,
			order:     idx,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].timestamp == items[j].timestamp {
			return items[i].order < items[j].order
		}
		return items[i].timestamp < items[j].timestamp
	})

	var buf bytes.Buffer
	for _, item := range items {
		buf.WriteString(item.raw)
		buf.WriteByte('\n')
	}

	normalized := buf.Bytes()
	changed := duplicatesRemoved > 0 || invalidRemoved > 0 || !bytes.Equal(normalized, data)
	return normalized, duplicatesRemoved, invalidRemoved, changed
}

func writeAtomic(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func inProgress(base string) bool {
	rebasePath := filepath.Join(base, ".git", "rebase-merge")
	mergePath := filepath.Join(base, ".git", "MERGE_HEAD")
	if _, err := os.Stat(rebasePath); err == nil {
		return true
	}
	if _, err := os.Stat(mergePath); err == nil {
		return true
	}
	return false
}

func backupTimestamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

func backupContinuum(base, ts string) (string, error) {
	backupPath := fmt.Sprintf("%s-backup-%s", base, ts)
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return "", err
	}
	if err := copyDir(base, backupPath); err != nil {
		return "", err
	}
	return backupPath, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		return copyFile(path, target, info)
	})
}

func copyFile(src, dst string, info os.FileInfo) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return nil
}

func InitRemote(remote string) error {
	if remote == "" {
		return fmt.Errorf("remote URL required")
	}

	base := ContinuumPath()
	empty, err := isDirEmpty(base)
	if err != nil {
		return err
	}
	if !empty {
		return fmt.Errorf("directory already exists. Use 'ctx sync --remote=%s' to add a remote to an existing installation.", remote)
	}
	if err := ensureDir(filepath.Dir(base)); err != nil {
		return fmt.Errorf("cannot create parent directory: %w", err)
	}
	git := vcs.NewGit(filepath.Join(base, "local", "git.log"))
	if err := git.Clone(remote, base); err != nil {
		return fmt.Errorf("cannot clone remote: %w", err)
	}
	for _, dir := range []string{
		filepath.Join(base, "local"),
		filepath.Join(base, "exports"),
		filepath.Join(base, "skills"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", dir, err)
		}
	}
	return nil
}

func isDirEmpty(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%s exists and is not a directory", path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func DeleteProject(project string) error {
	if err := ValidateProjectName(project); err != nil {
		return err
	}

	base := ContinuumPath()
	projectDir := filepath.Join(base, "projects", project)
	if _, err := os.Stat(projectDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project '%s' not found", project)
		}
		return fmt.Errorf("cannot stat project directory: %w", err)
	}
	rel := filepath.ToSlash(filepath.Join("projects", project))
	backupDir := filepath.Join(base, "local", ".delete-project-backup-"+project)
	_ = os.RemoveAll(backupDir)
	if err := os.Rename(projectDir, backupDir); err != nil {
		return fmt.Errorf("cannot prepare project deletion: %w", err)
	}
	if err := StageDeletedPaths([]string{rel}); err != nil {
		_ = os.Rename(backupDir, projectDir)
		return err
	}
	if err := events.Append(project, "", "project_deleted", "ok", "project removed"); err == nil {
		_ = StagePaths([]string{events.ActivityRelPath()})
	}
	msg := fmt.Sprintf("delete(%s): project removed", project)
	if err := CommitFiles(msg, nil); err != nil {
		_ = os.Rename(backupDir, projectDir)
		return err
	}
	_ = os.RemoveAll(backupDir)
	PushBestEffort()
	return nil
}

type gitRangeCounter interface {
	RevListCount(path string, args ...string) (int, error)
}

func markUnsynced(hash string) error {
	if hash == "" {
		return nil
	}
	path := unsyncedFilePath()
	existing := readLines(path)
	for _, h := range existing {
		if h == hash {
			return nil
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(hash + "\n")
	return err
}

func ClearUnsynced() error {
	path := unsyncedFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func UnsyncedCommits() []string {
	return readLines(unsyncedFilePath())
}

func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var lines []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func unsyncedFilePath() string {
	return filepath.Join(ContinuumPath(), unsyncedRelPath)
}

func logFilePath() string {
	return filepath.Join(ContinuumPath(), "local", "git.log")
}

func LastLogEntry() string {
	data, err := os.ReadFile(logFilePath())
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return ""
	}
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return ""
}
