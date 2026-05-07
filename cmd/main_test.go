package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseCaptureArgs(t *testing.T) {
	taskName, project, captureType, resolves, autoConfirm := parseCaptureArgs([]string{"my-task", "--project=my-project", "--yes"})
	if taskName != "my-task" {
		t.Fatalf("expected task my-task, got %q", taskName)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if captureType != "state" {
		t.Fatalf("expected default capture type state, got %q", captureType)
	}
	if resolves != "" {
		t.Fatalf("expected empty resolves, got %q", resolves)
	}
	if !autoConfirm {
		t.Fatal("expected autoConfirm to be true")
	}
}

func TestParseCaptureArgs_FlagBeforeTask(t *testing.T) {
	taskName, project, captureType, resolves, autoConfirm := parseCaptureArgs([]string{"--yes", "my-task"})
	if taskName != "my-task" {
		t.Fatalf("expected task my-task, got %q", taskName)
	}
	if project != "" {
		t.Fatalf("expected empty project, got %q", project)
	}
	if captureType != "state" {
		t.Fatalf("expected default capture type state, got %q", captureType)
	}
	if resolves != "" {
		t.Fatalf("expected empty resolves, got %q", resolves)
	}
	if !autoConfirm {
		t.Fatal("expected autoConfirm to be true")
	}
}

func TestParseCaptureArgs_TypeFlag(t *testing.T) {
	taskName, project, captureType, resolves, autoConfirm := parseCaptureArgs([]string{"my-task", "--project=my-project", "--type=request", "--resolves=proposal.20260416T095334Z.fdd7a5.md", "--yes"})
	if taskName != "my-task" {
		t.Fatalf("expected task my-task, got %q", taskName)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if captureType != "request" {
		t.Fatalf("expected capture type request, got %q", captureType)
	}
	if resolves != "proposal.20260416T095334Z.fdd7a5.md" {
		t.Fatalf("expected resolves filename, got %q", resolves)
	}
	if !autoConfirm {
		t.Fatal("expected autoConfirm to be true")
	}
}

func TestParseTaskArgs(t *testing.T) {
	taskName, project, autoConfirm := parseTaskArgs([]string{"--yes", "my-task", "--project=my-project"})
	if taskName != "my-task" {
		t.Fatalf("expected task my-task, got %q", taskName)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if !autoConfirm {
		t.Fatal("expected autoConfirm to be true")
	}
}

func TestParseContextArgsCompact(t *testing.T) {
	project, taskName, compact := parseContextArgs([]string{"my-task", "--project=my-project", "--compact"})
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if taskName != "my-task" {
		t.Fatalf("expected task my-task, got %q", taskName)
	}
	if !compact {
		t.Fatal("expected compact to be true")
	}
}

func TestParseArtifactListArgs(t *testing.T) {
	taskName, project, captureType := parseArtifactListArgs([]string{"--type=proposal", "my-task", "--project=my-project"})
	if taskName != "my-task" {
		t.Fatalf("expected task my-task, got %q", taskName)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if captureType != "proposal" {
		t.Fatalf("expected type proposal, got %q", captureType)
	}
}

func TestParseArtifactFileArgs(t *testing.T) {
	taskName, project, filename := parseArtifactFileArgs([]string{"my-task", "proposal.20260413T120000Z.aaaaaa.md", "--project=my-project"})
	if taskName != "my-task" {
		t.Fatalf("expected task my-task, got %q", taskName)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if filename != "proposal.20260413T120000Z.aaaaaa.md" {
		t.Fatalf("expected proposal filename, got %q", filename)
	}
}

func TestParseInitArgs(t *testing.T) {
	projectName, templatesPath, force, _ := parseInitArgs([]string{"--templates=./templates"})
	if projectName != "" {
		t.Fatalf("expected empty project, got %q", projectName)
	}
	if templatesPath != "./templates" {
		t.Fatalf("expected templates path ./templates, got %q", templatesPath)
	}
	if force {
		t.Fatal("expected force to be false")
	}
}

func TestParseInitArgs_OnlyTemplates(t *testing.T) {
	projectName, templatesPath, force, _ := parseInitArgs([]string{"--templates=/tmp/templates"})
	if projectName != "" {
		t.Fatalf("expected empty project, got %q", projectName)
	}
	if templatesPath != "/tmp/templates" {
		t.Fatalf("expected templates path /tmp/templates, got %q", templatesPath)
	}
	if force {
		t.Fatal("expected force to be false")
	}
}

func TestParseSyncArgs_RejectsUnexpectedArg(t *testing.T) {
	if _, _, _, err := parseSyncArgs([]string{"oops"}); err == nil {
		t.Fatal("expected sync usage error")
	}
}

func TestParseSyncArgs_WithPreferenceAndForce(t *testing.T) {
	remote, prefer, force, err := parseSyncArgs([]string{"--remote=git@example.com:ctx.git", "--prefer=remote", "--force"})
	if err != nil {
		t.Fatalf("parseSyncArgs: %v", err)
	}
	if remote != "git@example.com:ctx.git" {
		t.Fatalf("unexpected remote: %q", remote)
	}
	if prefer != "remote" {
		t.Fatalf("unexpected prefer: %q", prefer)
	}
	if !force {
		t.Fatal("expected force to be true")
	}
}

func TestParseSyncArgs_RejectsForceWithoutPreference(t *testing.T) {
	if _, _, _, err := parseSyncArgs([]string{"--force"}); err == nil {
		t.Fatal("expected sync usage error")
	}
}

func TestParseAgentInstallArgs_ProjectFlagOnly(t *testing.T) {
	project, force := parseAgentInstallArgs([]string{"--project=my-project", "--force"})
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if !force {
		t.Fatal("expected force to be true")
	}
}

func TestParseAgentProjectArgs(t *testing.T) {
	project, force, err := parseAgentProjectArgs([]string{"--project=my-project"})
	if err != nil {
		t.Fatalf("parseAgentProjectArgs: %v", err)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if force {
		t.Fatal("did not expect force")
	}
}

func TestParseAgentProjectArgs_AllowsEmptyProject(t *testing.T) {
	project, force, err := parseAgentProjectArgs(nil)
	if err != nil {
		t.Fatalf("parseAgentProjectArgs: %v", err)
	}
	if project != "" {
		t.Fatalf("expected empty project, got %q", project)
	}
	if force {
		t.Fatal("did not expect force")
	}
}

func TestParseAgentProjectArgs_AllowsForce(t *testing.T) {
	project, force, err := parseAgentProjectArgs([]string{"--project=my-project", "--force"})
	if err != nil {
		t.Fatalf("parseAgentProjectArgs: %v", err)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if !force {
		t.Fatal("expected force")
	}
}

func TestParseAgentProjectArgs_RejectsUnknownArg(t *testing.T) {
	if _, _, err := parseAgentProjectArgs([]string{"unexpected"}); err == nil {
		t.Fatal("expected usage error")
	}
}

func TestParseProjectCommandArgs(t *testing.T) {
	project, err := parseProjectCommandArgs([]string{"my-project"})
	if err != nil {
		t.Fatalf("parseProjectCommandArgs: %v", err)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
}

func TestParseProjectCommandArgs_RejectsFlagForm(t *testing.T) {
	if _, err := parseProjectCommandArgs([]string{"--project=my-project"}); err == nil {
		t.Fatal("expected parseProjectCommandArgs to reject --project flag")
	}
}

func TestParseProjectOnboardArgs_RejectsProjectFlagForm(t *testing.T) {
	if _, _, _, err := parseProjectOnboardArgs([]string{"--project=my-project", "--yes"}); err == nil {
		t.Fatal("expected parseProjectOnboardArgs to reject --project flag")
	}
}

func TestParseProjectOnboardArgs(t *testing.T) {
	project, force, autoConfirm, err := parseProjectOnboardArgs([]string{"my-project", "--force", "--yes"})
	if err != nil {
		t.Fatalf("parseProjectOnboardArgs: %v", err)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if !force {
		t.Fatal("expected force to be true")
	}
	if !autoConfirm {
		t.Fatal("expected autoConfirm to be true")
	}
}

func TestParseConfigSetArgs(t *testing.T) {
	key, value, err := parseConfigSetArgs([]string{"host", "mac125316"})
	if err != nil {
		t.Fatalf("parseConfigSetArgs: %v", err)
	}
	if key != "host" || value != "mac125316" {
		t.Fatalf("unexpected parse result: %q %q", key, value)
	}
}

func TestParseConfigSetArgs_RequiresTwoArgs(t *testing.T) {
	if _, _, err := parseConfigSetArgs([]string{"host"}); err == nil {
		t.Fatal("expected missing value error")
	}
}

func TestWatchArgs_ProjectAndInterval(t *testing.T) {
	project := ""
	interval := ""

	for _, arg := range []string{"--project=my-project", "--interval=5s"} {
		if val, ok := parseFlag(arg, "--project="); ok {
			project = val
		} else if val, ok := parseFlag(arg, "--interval="); ok {
			interval = val
		}
	}

	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if interval != "5s" {
		t.Fatalf("expected interval 5s, got %q", interval)
	}
}

func TestParseSearchArgs(t *testing.T) {
	project, taskName, query, limit, since, err := parseSearchArgsFull([]string{"--project=my-project", "--task=my-task", "--limit=5", "--since=24h", "build", "error"})
	if err != nil {
		t.Fatalf("parseSearchArgs: %v", err)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if taskName != "my-task" {
		t.Fatalf("expected task my-task, got %q", taskName)
	}
	if query != "build error" {
		t.Fatalf("expected query %q, got %q", "build error", query)
	}
	if limit != 5 {
		t.Fatalf("expected limit 5, got %d", limit)
	}
	if since != 24*time.Hour {
		t.Fatalf("expected since 24h, got %s", since)
	}
}

func TestParseSearchArgs_RequiresQuery(t *testing.T) {
	if _, _, _, _, _, err := parseSearchArgsFull([]string{"--project=my-project"}); err == nil {
		t.Fatal("expected missing query error")
	}
}

func TestParseSearchArgs_InvalidLimit(t *testing.T) {
	if _, _, _, _, _, err := parseSearchArgsFull([]string{"--limit=0", "error"}); err == nil {
		t.Fatal("expected invalid limit error")
	}
}

func TestParseSearchArgs_SinceDays(t *testing.T) {
	_, _, _, _, since, err := parseSearchArgsFull([]string{"--since=7d", "error"})
	if err != nil {
		t.Fatalf("parseSearchArgsFull: %v", err)
	}
	if since != 7*24*time.Hour {
		t.Fatalf("expected 7d, got %s", since)
	}
}

func TestParseHistoryArgs(t *testing.T) {
	project, taskName, limit, since, err := parseHistoryArgs([]string{"--project=my-project", "--task=my-task", "--limit=5", "--since=7d"})
	if err != nil {
		t.Fatalf("parseHistoryArgs: %v", err)
	}
	if project != "my-project" || taskName != "my-task" || limit != 5 || since != 7*24*time.Hour {
		t.Fatalf("unexpected parse result: %q %q %d %s", project, taskName, limit, since)
	}
}

func TestParseHistoryArgs_RejectsUnexpectedArgs(t *testing.T) {
	if _, _, _, _, err := parseHistoryArgs([]string{"oops"}); err == nil {
		t.Fatal("expected usage error")
	}
}

func TestParseDiffArgs(t *testing.T) {
	project, taskName, fromName, toName, err := parseDiffArgs([]string{"my-task", "snapshot.a.md", "snapshot.b.md", "--project=my-project"})
	if err != nil {
		t.Fatalf("parseDiffArgs: %v", err)
	}
	if project != "my-project" || taskName != "my-task" || fromName != "snapshot.a.md" || toName != "snapshot.b.md" {
		t.Fatalf("unexpected parse result: %q %q %q %q", project, taskName, fromName, toName)
	}
}

func TestParseDiffArgs_RequiresBothSnapshots(t *testing.T) {
	if _, _, _, _, err := parseDiffArgs([]string{"my-task", "snapshot.a.md"}); err == nil {
		t.Fatal("expected error for incomplete snapshot pair")
	}
}

func TestParseListArgs(t *testing.T) {
	project, status, err := parseListArgs([]string{"--project=my-project", "--status=closed"})
	if err != nil {
		t.Fatalf("parseListArgs: %v", err)
	}
	if project != "my-project" {
		t.Fatalf("expected project my-project, got %q", project)
	}
	if status != "closed" {
		t.Fatalf("expected status closed, got %q", status)
	}
}

func TestParseListArgs_All(t *testing.T) {
	_, status, err := parseListArgs([]string{"--all"})
	if err != nil {
		t.Fatalf("parseListArgs: %v", err)
	}
	if status != "all" {
		t.Fatalf("expected status all, got %q", status)
	}
}

func TestParseExportArgs_Task(t *testing.T) {
	projects, taskName, customPath, algo, session, err := parseExportArgs([]string{"my-task", "--project=my-project", "--path=/tmp/out", "--encrypt"})
	if err != nil {
		t.Fatalf("parseExportArgs: %v", err)
	}
	if len(projects) != 1 || projects[0] != "my-project" {
		t.Fatalf("unexpected projects: %#v", projects)
	}
	if taskName != "my-task" || customPath != "/tmp/out" || algo != "aes-gcm-v2" || session {
		t.Fatalf("unexpected parse result: %v %q %q %v", projects, taskName, customPath, session)
	}
}

func TestParseExportArgs_Projects(t *testing.T) {
	projects, taskName, _, _, session, err := parseExportArgs([]string{"--project=alpha,beta"})
	if err != nil {
		t.Fatalf("parseExportArgs: %v", err)
	}
	if taskName != "" || session {
		t.Fatalf("unexpected task/session result")
	}
	if len(projects) != 2 || projects[0] != "alpha" || projects[1] != "beta" {
		t.Fatalf("unexpected projects: %#v", projects)
	}
}

func TestParseExportArgs_Session(t *testing.T) {
	projects, taskName, _, _, session, err := parseExportArgs([]string{"--session"})
	if err != nil {
		t.Fatalf("parseExportArgs: %v", err)
	}
	if !session || taskName != "" || len(projects) != 0 {
		t.Fatalf("unexpected parse result")
	}
}

func TestIsVersionCommand(t *testing.T) {
	if !isVersionCommand("--version") {
		t.Fatal("expected --version to be recognized")
	}
	if !isVersionCommand("version") {
		t.Fatal("expected version subcommand to be recognized")
	}
	if isVersionCommand("watch") {
		t.Fatal("did not expect watch to be recognized as version")
	}
}

func TestPrintVersion(t *testing.T) {
	oldVersion := version
	oldCommit := commit
	version = "v1.2.3-test"
	commit = "none"
	t.Cleanup(func() {
		version = oldVersion
		commit = oldCommit
	})

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	printVersion()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if got := buf.String(); got != "ctx v1.2.3-test\n" {
		t.Fatalf("unexpected version output: %q", got)
	}
}

func TestVersionString_UsesInjectedReleaseVersion(t *testing.T) {
	oldVersion := version
	oldCommit := commit
	version = "v1.2.3"
	commit = "abcdef1234567890"
	t.Cleanup(func() {
		version = oldVersion
		commit = oldCommit
	})

	if got := versionString(); got != "v1.2.3" {
		t.Fatalf("expected release version, got %q", got)
	}
}

func TestVersionString_UsesInjectedCommitForDev(t *testing.T) {
	oldVersion := version
	oldCommit := commit
	version = "dev"
	commit = "abcdef1234567890"
	t.Cleanup(func() {
		version = oldVersion
		commit = oldCommit
	})

	if got := versionString(); got != "dev+abcdef1" {
		t.Fatalf("expected dev commit version, got %q", got)
	}
}

func TestShortCommit(t *testing.T) {
	if got := shortCommit("abcdef123"); got != "abcdef1" {
		t.Fatalf("unexpected short commit: %q", got)
	}
	if got := shortCommit("abc"); got != "abc" {
		t.Fatalf("unexpected short commit: %q", got)
	}
}

func TestStyleUsageInline_NoColorWhenDisabled(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "xterm-256color")
	got := styleUsageInline("ctx init --force")
	if got != "ctx init --force" {
		t.Fatalf("styleUsageInline() = %q, want unchanged text", got)
	}
}

func TestStyleUsageInline_HighlightsCommandsWhenEnabled(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	oldEnabled := usageColorsEnabled
	usageColorsEnabled = func() bool { return true }
	defer func() { usageColorsEnabled = oldEnabled }()

	got := styleUsageInline("run ctx init --force now")
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected ANSI styling, got %q", got)
	}
	if !strings.Contains(got, "ctx init") || !strings.Contains(got, "--force") {
		t.Fatalf("expected command tokens to remain visible, got %q", got)
	}
}

func TestPadUsageColumn_UsesVisibleWidth(t *testing.T) {
	oldEnabled := usageColorsEnabled
	usageColorsEnabled = func() bool { return true }
	defer func() { usageColorsEnabled = oldEnabled }()

	styled := styleUsageInline("ctx init")
	padded := padUsageColumn(styled, 12)
	if got := visibleWidth(padded); got != 12 {
		t.Fatalf("visibleWidth(padded) = %d, want %d", got, 12)
	}
}

func TestStyleUsageInline_HighlightsFullChoiceFlag(t *testing.T) {
	oldEnabled := usageColorsEnabled
	usageColorsEnabled = func() bool { return true }
	defer func() { usageColorsEnabled = oldEnabled }()

	got := styleUsageInline("--status=<active|closed>")
	if !strings.Contains(got, "active|closed") {
		t.Fatalf("expected full choice flag to remain visible, got %q", got)
	}
	if visibleWidth(got) != len("--status=<active|closed>") {
		t.Fatalf("unexpected visible width for styled flag: %d", visibleWidth(got))
	}
}
