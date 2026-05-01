package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"continuum/internal/events"
)

func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func TestSyncAddsRemote(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	remote := t.TempDir()

	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}

	if out, err := exec.Command("git", "-C", base, "remote", "add", "origin", remote).CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", base, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v\n%s", err, out)
	}
	if err := exec.Command("git", "-C", base, "remote", "remove", "origin").Run(); err != nil {
		t.Fatalf("git remote remove failed: %v", err)
	}

	result, err := Sync(remote)
	if err != nil {
		t.Fatalf("Sync() error: %v", err)
	}
	if !result.RemoteAdded {
		t.Fatal("expected remote to be added")
	}

	out, err := exec.Command("git", "-C", base, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		t.Fatalf("git remote get-url failed: %v\n%s", err, out)
	}
	got := strings.TrimSpace(string(out))
	if filepath.Clean(got) != filepath.Clean(remote) {
		t.Fatalf("remote URL mismatch: expected %s, got %s", remote, got)
	}
}

func TestInitRemote_ClonesWithoutLocalCommit(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	remote := t.TempDir()
	seed := t.TempDir()

	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "clone", remote, seed).CombinedOutput(); err != nil {
		t.Fatalf("git clone seed failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", seed, "config", "user.email", "seed@test.com").CombinedOutput(); err != nil {
		t.Fatalf("git config email failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", seed, "config", "user.name", "Seed").CombinedOutput(); err != nil {
		t.Fatalf("git config name failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(seed, "profile.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	if out, err := exec.Command("git", "-C", seed, "add", "--", "profile.md").CombinedOutput(); err != nil {
		t.Fatalf("git add seed file failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", seed, "commit", "-m", "init: seed remote").CombinedOutput(); err != nil {
		t.Fatalf("git commit seed file failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", seed, "push", "origin", "HEAD:main").CombinedOutput(); err != nil {
		t.Fatalf("git push seed file failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", remote, "symbolic-ref", "HEAD", "refs/heads/main").CombinedOutput(); err != nil {
		t.Fatalf("git symbolic-ref HEAD failed: %v\n%s", err, out)
	}

	if err := InitRemote(remote); err != nil {
		t.Fatalf("InitRemote() error: %v", err)
	}

	out, err := exec.Command("git", "-C", base, "status", "--short").CombinedOutput()
	if err != nil {
		t.Fatalf("git status failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("expected clean worktree after init --remote, got:\n%s", out)
	}

	out, err = exec.Command("git", "-C", base, "rev-list", "--count", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "1" {
		t.Fatalf("expected cloned history only, got %q commits", strings.TrimSpace(string(out)))
	}
}

func TestSyncBootstrapsEmptyRemote(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	remote := t.TempDir()

	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}

	result, err := Sync(remote)
	if err != nil {
		t.Fatalf("Sync() error: %v", err)
	}
	if !result.RemoteAdded {
		t.Fatal("expected remote to be added")
	}
	if result.PullCount != 0 {
		t.Fatalf("expected zero pulled commits, got %d", result.PullCount)
	}
	if result.PushCount == 0 {
		t.Fatal("expected initial commits to be pushed to empty remote")
	}

	out, err := exec.Command("git", "-C", remote, "rev-parse", "--verify", "main").CombinedOutput()
	if err != nil {
		t.Fatalf("expected main branch to exist on remote: %v\n%s", err, out)
	}

	out, err = exec.Command("git", "-C", base, "ls-remote", "--heads", "origin", "main").CombinedOutput()
	if err != nil {
		t.Fatalf("git ls-remote failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) == "" {
		t.Fatal("expected origin/main to be visible after sync")
	}
}

func TestSyncCountsPulledAndPushedCommits(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	remote := t.TempDir()
	peer := t.TempDir()

	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}

	if out, err := exec.Command("git", "-C", base, "remote", "add", "origin", remote).CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", base, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v\n%s", err, out)
	}

	if out, err := exec.Command("git", "clone", "--branch", "main", remote, peer).CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "config", "user.email", "test@test.com").CombinedOutput(); err != nil {
		t.Fatalf("git config email failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "config", "user.name", "Test").CombinedOutput(); err != nil {
		t.Fatalf("git config name failed: %v\n%s", err, out)
	}

	if err := os.WriteFile(filepath.Join(peer, "remote-note.txt"), []byte("remote"), 0o644); err != nil {
		t.Fatalf("write remote file: %v", err)
	}
	if out, err := exec.Command("git", "-C", peer, "add", "--", "remote-note.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add remote file failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "commit", "-m", "test: remote change").CombinedOutput(); err != nil {
		t.Fatalf("git commit remote file failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push remote file failed: %v\n%s", err, out)
	}

	if err := os.WriteFile(filepath.Join(base, "local-note.txt"), []byte("local"), 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}
	if err := CommitFiles("test: local change", []string{"local-note.txt"}); err != nil {
		t.Fatalf("CommitFiles() error: %v", err)
	}

	result, err := Sync("")
	if err != nil {
		t.Fatalf("Sync() error: %v", err)
	}
	if result.PullCount != 1 {
		t.Fatalf("expected one pulled commit, got %d", result.PullCount)
	}
	if result.PushCount != 1 {
		t.Fatalf("expected one pushed commit, got %d", result.PushCount)
	}
	if result.Bootstrapped {
		t.Fatal("did not expect bootstrap on populated remote")
	}
	if result.LogEntry != "" {
		t.Fatalf("did not expect log entry on successful sync, got %q", result.LogEntry)
	}
}

func TestSyncRejectsDirtyWorktree(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	remote := t.TempDir()

	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}
	if out, err := exec.Command("git", "-C", base, "remote", "add", "origin", remote).CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", base, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v\n%s", err, out)
	}

	if err := os.WriteFile(filepath.Join(base, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	_, err := Sync("")
	if err == nil {
		t.Fatal("expected dirty-worktree sync error")
	}
	if got := err.Error(); !strings.Contains(got, "Local Continuum storage has uncommitted changes:") {
		t.Fatalf("unexpected sync error: %q", got)
	}
}

func TestSyncPreferLocalCommitsDirtyWorktree(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	remote := t.TempDir()

	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}
	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}
	if out, err := exec.Command("git", "-C", base, "remote", "add", "origin", remote).CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", base, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(base, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	result, err := SyncWithOptions(SyncOptions{Prefer: "local", Force: true})
	if err != nil {
		t.Fatalf("SyncWithOptions() error: %v", err)
	}
	if result.PushCount == 0 {
		t.Fatal("expected local preservation commit to be pushed")
	}

	out, err := exec.Command("git", "-C", base, "status", "--short").CombinedOutput()
	if err != nil {
		t.Fatalf("git status failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("expected clean worktree after prefer=local sync, got:\n%s", out)
	}

	out, err = exec.Command("git", "-C", base, "log", "--format=%s", "-2").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "sync: preserve local continuum changes") {
		t.Fatalf("expected preservation commit in recent log, got:\n%s", out)
	}
}

func TestSyncPreferLocalOverridesRemoteState(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	remote := t.TempDir()
	peer := t.TempDir()

	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}
	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}
	if out, err := exec.Command("git", "-C", base, "remote", "add", "origin", remote).CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", base, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "clone", "--branch", "main", remote, peer).CombinedOutput(); err != nil {
		t.Fatalf("git clone peer failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "config", "user.email", "peer@test.com").CombinedOutput(); err != nil {
		t.Fatalf("git config email failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "config", "user.name", "Peer").CombinedOutput(); err != nil {
		t.Fatalf("git config name failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(peer, "remote-note.txt"), []byte("remote"), 0o644); err != nil {
		t.Fatalf("write remote file: %v", err)
	}
	if out, err := exec.Command("git", "-C", peer, "add", "--", "remote-note.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add remote file failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "commit", "-m", "test: remote change").CombinedOutput(); err != nil {
		t.Fatalf("git commit remote file failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push remote file failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(base, "local-only.txt"), []byte("local commit"), 0o644); err != nil {
		t.Fatalf("write local-only file: %v", err)
	}
	if err := CommitFiles("test: local-only change", []string{"local-only.txt"}); err != nil {
		t.Fatalf("CommitFiles() error: %v", err)
	}

	result, err := SyncWithOptions(SyncOptions{Prefer: "local", Force: true})
	if err != nil {
		t.Fatalf("SyncWithOptions() error: %v", err)
	}
	if result.PushCount == 0 {
		t.Fatal("expected local state to be pushed over remote")
	}
	if _, err := os.Stat(filepath.Join(base, "remote-note.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected remote-only file to stay out of local state, got err=%v", err)
	}

	verify := t.TempDir()
	if out, err := exec.Command("git", "clone", "--branch", "main", remote, verify).CombinedOutput(); err != nil {
		t.Fatalf("git clone verify failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(verify, "local-only.txt")); err != nil {
		t.Fatalf("expected local-only file to be pushed to remote: %v", err)
	}
	if _, err := os.Stat(filepath.Join(verify, "remote-note.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected remote-only file to be removed from remote after prefer=local, got err=%v", err)
	}
}

func TestSyncPreferRemoteDiscardsDirtyWorktree(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	remote := t.TempDir()
	peer := t.TempDir()

	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}
	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}
	if out, err := exec.Command("git", "-C", base, "remote", "add", "origin", remote).CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", base, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "clone", "--branch", "main", remote, peer).CombinedOutput(); err != nil {
		t.Fatalf("git clone peer failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "config", "user.email", "peer@test.com").CombinedOutput(); err != nil {
		t.Fatalf("git config email failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "config", "user.name", "Peer").CombinedOutput(); err != nil {
		t.Fatalf("git config name failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(peer, "remote-note.txt"), []byte("remote"), 0o644); err != nil {
		t.Fatalf("write remote file: %v", err)
	}
	if out, err := exec.Command("git", "-C", peer, "add", "--", "remote-note.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add remote file failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "commit", "-m", "test: remote change").CombinedOutput(); err != nil {
		t.Fatalf("git commit remote file failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", peer, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push remote file failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(base, "local-only.txt"), []byte("local commit"), 0o644); err != nil {
		t.Fatalf("write local-only file: %v", err)
	}
	if err := CommitFiles("test: local-only change", []string{"local-only.txt"}); err != nil {
		t.Fatalf("CommitFiles() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	result, err := SyncWithOptions(SyncOptions{Prefer: "remote", Force: true})
	if err != nil {
		t.Fatalf("SyncWithOptions() error: %v", err)
	}
	if result.PushCount != 0 {
		t.Fatalf("expected no pushed commits after prefer=remote discard, got %d", result.PushCount)
	}

	if _, err := os.Stat(filepath.Join(base, "dirty.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected dirty file to be discarded, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "local-only.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected local-only file to be discarded, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "remote-note.txt")); err != nil {
		t.Fatalf("expected remote file to be present after prefer=remote sync: %v", err)
	}
}

func TestSyncPreferRemoteResetsCleanAheadLocalState(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	remote := t.TempDir()

	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}
	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}
	if out, err := exec.Command("git", "-C", base, "remote", "add", "origin", remote).CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", base, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(base, "local-only.txt"), []byte("local commit"), 0o644); err != nil {
		t.Fatalf("write local-only file: %v", err)
	}
	if err := CommitFiles("test: clean ahead local change", []string{"local-only.txt"}); err != nil {
		t.Fatalf("CommitFiles() error: %v", err)
	}

	result, err := SyncWithOptions(SyncOptions{Prefer: "remote", Force: true})
	if err != nil {
		t.Fatalf("SyncWithOptions() error: %v", err)
	}
	if result.PushCount != 0 {
		t.Fatalf("expected no pushed commits after prefer=remote reset, got %d", result.PushCount)
	}
	if _, err := os.Stat(filepath.Join(base, "local-only.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected local-only file to be discarded, got err=%v", err)
	}
}

func TestRepairNoIssues(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	withTempContinuum(t)
	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}

	msg, err := Repair()
	if err != nil {
		t.Fatalf("Repair() error: %v", err)
	}
	if msg != "No issues detected." {
		t.Fatalf("unexpected repair message: %s", msg)
	}

	items, _, err := events.ReadFromOffset(0)
	if err != nil {
		t.Fatalf("ReadFromOffset: %v", err)
	}
	if len(items) == 0 || items[len(items)-1].Type != "repair" {
		t.Fatalf("expected trailing repair event, got %#v", items)
	}
}

func TestRepairActivityLog_DeduplicatesAndSorts(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}

	lines := []string{
		`{"timestamp":"2026-03-31T10:00:02Z","agent":"a","host":"h","type":"capture","status":"ok"}`,
		`{"timestamp":"2026-03-31T10:00:01Z","agent":"b","host":"h","type":"capture","status":"ok"}`,
		`{"timestamp":"2026-03-31T10:00:02Z","agent":"a","host":"h","type":"capture","status":"ok"}`,
		`<<<<<<< HEAD`,
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.MkdirAll(filepath.Join(base, "events"), 0o755); err != nil {
		t.Fatalf("mkdir events: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, events.ActivityRelPath()), []byte(content), 0o644); err != nil {
		t.Fatalf("write activity log: %v", err)
	}

	msg, err := RepairActivityLog()
	if err != nil {
		t.Fatalf("RepairActivityLog() error: %v", err)
	}
	if msg != "Activity log repaired: 1 duplicates removed." {
		t.Fatalf("unexpected activity repair message: %q", msg)
	}

	data, err := os.ReadFile(filepath.Join(base, events.ActivityRelPath()))
	if err != nil {
		t.Fatalf("read repaired activity log: %v", err)
	}
	expected := strings.Join([]string{
		`{"timestamp":"2026-03-31T10:00:01Z","agent":"b","host":"h","type":"capture","status":"ok"}`,
		`{"timestamp":"2026-03-31T10:00:02Z","agent":"a","host":"h","type":"capture","status":"ok"}`,
		"",
	}, "\n")
	if string(data) != expected {
		t.Fatalf("unexpected repaired activity log:\n%s", data)
	}

	out, err := exec.Command("git", "-C", base, "log", "--format=%s", "-1").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "repair: normalize activity log" {
		t.Fatalf("unexpected activity repair commit message: %q", strings.TrimSpace(string(out)))
	}
}

func TestRepairActivityLog_Clean(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	withTempContinuum(t)
	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}

	msg, err := RepairActivityLog()
	if err != nil {
		t.Fatalf("RepairActivityLog() error: %v", err)
	}
	if msg != "Activity log is clean." {
		t.Fatalf("unexpected clean activity repair message: %q", msg)
	}
}

func TestResumeLocalOnlyWithoutRemote(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	withTempContinuum(t)
	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}

	result, err := Resume()
	if err != nil {
		t.Fatalf("Resume() error: %v", err)
	}
	if result.Sync != nil {
		t.Fatal("expected no sync result without remote")
	}
	if result.SyncWarning != "remote not configured; continuing in local-only mode" {
		t.Fatalf("unexpected sync warning: %q", result.SyncWarning)
	}
}

func TestResumeRejectsDirtyWorktree(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	_, err := Resume()
	if err == nil {
		t.Fatal("expected dirty-worktree resume error")
	}
	if got := err.Error(); !strings.Contains(got, "Continuum storage has local changes.") {
		t.Fatalf("unexpected resume error: %q", got)
	}
}

func TestResumeSyncsWhenRemoteConfigured(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}

	base := withTempContinuum(t)
	remote := t.TempDir()

	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	if err := InitSession(false); err != nil {
		t.Fatalf("InitSession() error: %v", err)
	}
	if out, err := exec.Command("git", "-C", base, "remote", "add", "origin", remote).CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v\n%s", err, out)
	}

	result, err := Resume()
	if err != nil {
		t.Fatalf("Resume() error: %v", err)
	}
	if result.Sync == nil {
		t.Fatal("expected sync result with remote configured")
	}
	if !result.Sync.RemoteAdded && !result.Sync.Bootstrapped && result.Sync.PushCount == 0 {
		t.Fatal("expected resume to perform initial sync work")
	}
	if result.SyncWarning != "" {
		t.Fatalf("unexpected sync warning: %q", result.SyncWarning)
	}
}
