package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"continuum/internal/prompt"
)

var (
	usageCommandPattern = regexp.MustCompile("(`[^`]+`|ctx\\s+[^\\s].*|--[A-Za-z0-9][A-Za-z0-9<>=,./:_|\\-\\[\\]]*)")
	usageInlinePattern  = regexp.MustCompile("(`[^`]+`|--[A-Za-z0-9][A-Za-z0-9<>=,./:_|\\-\\[\\]]*)")
	usageTitleStyle     = ansiStyle("1", "38;5;73")
	usageSectionStyle   = ansiStyle("1", "38;5;110")
	usageCommandStyle   = ansiStyle("38;5;221")
	usageResetStyle     = "\x1b[0m"
	usageColorsEnabled  = defaultUsageColorsEnabled
)

func printUsage() {
	fmt.Println(styleUsageTitle("Continuum"))
	fmt.Println("Context orchestration for AI-assisted development")
	fmt.Println()
	fmt.Println(styleUsageSection("Usage:"))
	fmt.Println("  " + styleUsageInline("ctx <command> [options]"))
	fmt.Println()

	printUsageSection("Setup", [][2]string{
		{"ctx init", "initialize the local Continuum session"},
		{"ctx init --remote=<url>", "clone Continuum storage from a remote repo"},
		{"ctx project list", "list all projects"},
		{"ctx project init <project>", "create a project inside Continuum storage"},
		{"ctx project onboard <project> [--force] [--yes]", "save streamed project context for an existing codebase"},
		{"ctx project delete <project> [--yes]", "remove a project and commit the deletion"},
		{"ctx config set host <name>", "persist a local machine identity for events and commit metadata"},
		{"ctx agent install --project=<name> [--force]", "inject or re-inject bootstrap instructions into agent files"},
		{"ctx agent status [--project=<name>]", "check whether installed bootstrap instructions are current"},
		{"ctx agent update [--project=<name>] [--force]", "refresh templates and re-inject stale bootstrap instructions"},
		{"ctx agent remove", "remove injected bootstrap instructions"},
	})

	printUsageSection("Context", [][2]string{
		{"ctx resume", "prepare Continuum storage for a new work session"},
		{"ctx context", "print the current project context"},
		{"ctx context <task> [--compact]", "print context for a specific task"},
		{"ctx capture <task> --project=<name> [--type=state|proposal|request|response|decision] [--resolves=<filename>]", "save task state or a collaboration artifact"},
		{"ctx history", "tell the project or task story from snapshots and handoffs"},
		{"ctx timeline", "print the raw chronological activity timeline"},
		{"ctx search <query>", "search snapshots, handoffs, and collaboration artifacts"},
		{"ctx artifact list <task>", "list proposal/request/response/decision artifacts"},
		{"ctx artifact show <task> <filename>", "print a collaboration artifact"},
		{"ctx resolve <task> <filename>", "mark a collaboration artifact resolved"},
		{"ctx diff <task>", "diff the latest two snapshots for a task"},
		{"ctx sync [--remote=<url>] [--prefer=local|remote] [--force]", "pull and push Continuum git history"},
		{"ctx watch [--project=<name>] [--interval=<duration>] [--tui]", "stream live Continuum activity events"},
		{"ctx repair [--activity]", "validate and recover local git storage or normalize the activity log"},
	})

	printUsageSection("Task Management", [][2]string{
		{"ctx list", "list active tasks in the current project"},
		{"ctx task start <task> --project=<name>", "create a new task"},
		{"ctx task close <task> --project=<name>", "mark a task as closed"},
		{"ctx task reopen <task> --project=<name>", "mark a task as active again"},
		{"ctx task delete <task> --project=<name> [--yes]", "remove task state and commit the deletion"},
		{"ctx handoff <task> --project=<name>", "save handoff notes for a task"},
		{"ctx snapshot refresh <task> --project=<name>", "write a new snapshot revision for a task"},
		{"ctx snapshot clean <task> --project=<name>", "prune older snapshots for a task"},
	})

	printUsageSection("Skills", [][2]string{
		{"ctx skill list", "list all reusable skills"},
		{"ctx skill show <name>", "print a skill's content"},
		{"ctx skill save <name> [--description=<text>] [--yes]", "save streamed content as a named skill"},
		{"ctx skill delete <name> [--yes]", "delete a skill and remove it from the index"},
	})

	printUsageSection("Import / Export", [][2]string{
		{"ctx export <task>", "export a task archive"},
		{"ctx export --project=<name>", "export one or more project archives"},
		{"ctx export --session", "export the full Continuum session"},
		{"ctx import <path>", "import an exported archive"},
	})

	printUsageSection("Options", [][2]string{
		{"--project=<name>", "explicit project name; defaults to current directory"},
		{"--session", "export the full Continuum session"},
		{"--compact", "print a token-efficient context digest"},
		{"--task=<name>", "restrict search results to a specific task"},
		{"--status=<active|closed>", "filter task list by status"},
		{"--type=state|proposal|request|response|decision", "choose what ctx capture writes; default is state"},
		{"--resolves=<filename>", "link a decision capture to a proposal/request and resolve it"},
		{"--all", "show tasks with any status in list output"},
		{"--limit=<n>", "limit search output to the newest N matches"},
		{"--since=<duration>", "restrict search results to recent history (e.g. 24h, 7d)"},
		{"--templates=<path>", "template source directory for init"},
		{"--force", "overwrite existing init/template seed files or re-inject agent bootstrap"},
		{"--prefer=local|remote", "when sync finds local changes, preserve them or discard them before syncing"},
		{"--path=<path>", "custom export/import destination"},
		{"--encrypt[=<algo>]", "passphrase protection for exported archives (default: aes-gcm-v2)"},
		{"--decrypt[=<algo>]", "decrypt an imported archive (aes-gcm-v2)"},
	})

	printUsageSection("Environment", [][2]string{
		{"CONTINUUM_PATH", "Continuum storage location (default: ~/.ctx)"},
		{"CONTINUUM_HOST", "override machine identity for events and commit metadata"},
	})

	printUsageSection("Examples", [][2]string{
		{"ctx project init my-project", "create a project inside existing Continuum storage"},
		{"ctx project onboard my-project --yes < generated-project-context.md", "save generated project context into Continuum"},
		{"ctx init --remote=<url>", "clone Continuum storage from a remote"},
		{"ctx agent install --project=my-project --force", "re-inject updated bootstrap instructions"},
		{"ctx agent status --project=my-project", "check installed bootstrap freshness"},
		{"ctx agent update --project=my-project", "refresh stale agent instructions"},
		{"ctx resume", "repair, sync, and orient before choosing a project"},
		{"ctx sync --remote=<url>", "attach a remote and bootstrap it if empty"},
		{"ctx sync --prefer=local", "confirm and preserve local changes before syncing"},
		{"ctx sync --force --prefer=remote", "discard local changes without prompting and resync"},
		{"ctx --version", "print the installed ctx version"},
		{"ctx context --project=my-project", "show the current project context"},
		{"ctx context my-task --project=my-project --compact", "show a compact task digest"},
		{"ctx history --project=my-project --since=7d", "tell the recent project story"},
		{"ctx timeline --project=my-project --limit=20", "print the recent raw activity timeline"},
		{"ctx search --project=my-project --limit=5 --since=7d error", "search task history for a keyword"},
		{"ctx artifact list my-task --project=my-project", "list collaboration artifacts for a task"},
		{"ctx artifact show my-task proposal.20260413T114345Z.441b2c.md --project=my-project", "read a collaboration artifact"},
		{"ctx resolve my-task proposal.20260413T114345Z.441b2c.md --project=my-project", "remove an artifact from open counts"},
		{"ctx diff my-task --project=my-project", "diff the last two snapshots for a task"},
		{"ctx capture my-task --project=my-project --yes", "save a snapshot non-interactively"},
		{"ctx capture my-task --project=my-project --type=proposal --yes", "save a collaboration proposal without changing task state"},
		{"ctx capture my-task --project=my-project --type=decision --resolves=proposal.20260413T114345Z.441b2c.md --yes", "save a decision and resolve a linked proposal"},
		{"ctx handoff my-task --project=my-project --yes", "save a handoff non-interactively"},
		{"ctx snapshot refresh my-task --project=my-project --yes", "save a refreshed snapshot"},
		{"ctx export my-task --encrypt", "export an encrypted task archive"},
		{"ctx export --project=app,docs --encrypt", "export encrypted archives for one or more projects"},
		{"ctx export --session --encrypt", "export an encrypted full-session archive"},
	})

	fmt.Println(styleUsageSection("Security note:"))
	fmt.Println("  Export encryption uses aes-gcm-v2 with Argon2id key derivation.")
	fmt.Println("  Passphrases are read from stdin; interactive entry is safer than piping literals.")
}

func printUsageSection(title string, rows [][2]string) {
	if len(rows) == 0 {
		return
	}
	fmt.Printf("%s\n", styleUsageSection(title+":"))
	width := 0
	for _, row := range rows {
		if len(row[0]) > width {
			width = len(row[0])
		}
	}
	for _, row := range rows {
		fmt.Printf("  %s  %s\n", padUsageColumn(styleUsageCommand(row[0]), width), styleUsageInline(row[1]))
	}
	fmt.Println()
}

func styleUsageTitle(value string) string {
	if !usageColorsEnabled() {
		return value
	}
	return usageTitleStyle + value + usageResetStyle
}

func styleUsageSection(value string) string {
	if !usageColorsEnabled() {
		return value
	}
	return usageSectionStyle + value + usageResetStyle
}

func styleUsageCommand(value string) string {
	if !usageColorsEnabled() {
		return value
	}
	return usageCommandPattern.ReplaceAllStringFunc(value, func(match string) string {
		return usageCommandStyle + match + usageResetStyle
	})
}

func styleUsageInline(value string) string {
	if !usageColorsEnabled() {
		return value
	}
	return usageInlinePattern.ReplaceAllStringFunc(value, func(match string) string {
		return usageCommandStyle + match + usageResetStyle
	})
}

func defaultUsageColorsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if term := os.Getenv("TERM"); term == "" || term == "dumb" {
		return false
	}
	return prompt.IsInteractiveOutput()
}

func ansiStyle(codes ...string) string {
	return prompt.AnsiStyle(codes...)
}

func padUsageColumn(value string, width int) string {
	visible := prompt.VisibleWidth(value)
	if visible >= width {
		return value
	}
	return value + strings.Repeat(" ", width-visible)
}
