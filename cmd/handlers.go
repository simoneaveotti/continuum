package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"continuum/internal/agent"
	"continuum/internal/context"
	"continuum/internal/export"
	"continuum/internal/filestore"
	"continuum/internal/history"
	"continuum/internal/identity"
	"continuum/internal/prompt"
	"continuum/internal/search"
	"continuum/internal/setup"
	"continuum/internal/skill"
	"continuum/internal/task"
	"continuum/internal/template"
)

// die prints an error to stderr and exits with code 1.
func die(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}

// dieUsage prints usage message(s) to stderr and exits with code 1.
func dieUsage(lines ...string) {
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, line)
	}
	os.Exit(1)
}

func resolveProject(project string) string {
	if project != "" {
		return project
	}
	detected, err := setup.DetectProject()
	if err != nil {
		die(err)
	}
	return detected
}

func resolveAgentProject(command, project string) string {
	if project != "" {
		return project
	}
	detected, err := setup.DetectProject()
	if err != nil {
		die(fmt.Errorf("project not specified and no local project is configured.\nRun:\n  ctx agent %s --project=<name>", command))
	}
	return detected
}

func handleInit(args []string) {
	projectName, templatesPath, force, remote := parseInitArgs(args)
	if templatesPath != "" {
		if err := template.ValidateSourcePath(templatesPath); err != nil {
			die(err)
		}
		template.SetSourcePath(templatesPath)
	}
	if projectName != "" {
		dieUsage(initUsage)
	}

	if remote != "" {
		if err := setup.InitRemote(remote); err != nil {
			die(err)
		}
		return
	}

	if err := setup.InitSession(force); err != nil {
		die(err)
	}
	fmt.Println("Session initialized.")
	fmt.Println("Templates: .ctx/templates/")
	fmt.Println("Run 'ctx project init <project>' to add a project.")
}

func handleProjectList() {
	projects, err := setup.ListProjects()
	if err != nil {
		die(err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return
	}

	fmt.Println("Projects:")
	for _, p := range projects {
		fmt.Printf("  - %s\n", p)
	}
}

func handleCapture(args []string) {
	if len(args) < 1 {
		dieUsage(captureUsage)
	}
	taskName, project, captureTypeValue, resolves, autoConfirm := parseCaptureArgs(args)
	project = resolveProject(project)
	captureType, err := filestore.ValidateCaptureType(captureTypeValue)
	if err != nil {
		die(err)
	}
	if err := task.CaptureWithOptions(taskName, project, task.CaptureOptions{
		Type:        captureType,
		AutoConfirm: autoConfirm,
		Resolves:    resolves,
	}); err != nil {
		die(err)
	}
}

func handleContext(args []string) {
	project, taskName, compact := parseContextArgs(args)
	project = resolveProject(project)

	var output string
	if taskName == "" {
		ctxData, err := context.LoadFullContext(project)
		if err != nil {
			die(err)
		}
		if compact {
			output = context.BuildCompactContextPackage(ctxData, "", project)
		} else {
			output = context.BuildContextPackage(ctxData, "", project)
		}
	} else {
		ctxData, err := context.Load(taskName, project)
		if err != nil {
			die(err)
		}
		if compact {
			output = context.BuildCompactContextPackage(ctxData, taskName, project)
		} else {
			output = context.BuildContextPackage(ctxData, taskName, project)
		}
	}

	fmt.Println(output)
}

func handleSync(args []string) {
	remote, prefer, force, err := parseSyncArgs(args)
	if err != nil {
		die(err)
	}

	if prefer != "" && !force {
		confirmed, err := confirmSyncPreference(prefer)
		if err != nil {
			die(err)
		}
		if !confirmed {
			fmt.Println("Sync canceled.")
			return
		}
		force = true
	}

	result, err := setup.SyncWithOptions(setup.SyncOptions{
		Remote: remote,
		Prefer: prefer,
		Force:  force,
	})
	if err != nil {
		die(err)
	}
	if result.RemoteAdded {
		fmt.Println("Remote origin configured.")
	}
	if result.Bootstrapped {
		fmt.Println("Empty remote initialized with branch main.")
	}
	switch result.Preference {
	case "local":
		fmt.Println("Sync completed.")
		fmt.Println("Strategy: prefer local")
		fmt.Printf("Published local commits: %d\n", result.LocalAheadBefore)
		fmt.Printf("Discarded remote-only commits: %d\n", result.RemoteAheadBefore)
	case "remote":
		fmt.Println("Sync completed.")
		fmt.Println("Strategy: prefer remote")
		fmt.Printf("Kept remote commits: %d\n", result.RemoteAheadBefore)
		fmt.Printf("Discarded local-only commits: %d\n", result.LocalAheadBefore)
	default:
		fmt.Printf("Sync completed. Push: %d commit(s). Pull: %d commit(s).\n", result.PushCount, result.PullCount)
	}
	if result.LogEntry != "" {
		fmt.Println("Log git:", result.LogEntry)
	}
}

func confirmSyncPreference(prefer string) (bool, error) {
	message := "This will preserve local uncommitted Continuum changes and sync them to the remote. Continue? [y/N]: "
	if prefer == "remote" {
		message = "This will discard local uncommitted Continuum changes and resync from the remote. Continue? [y/N]: "
	}
	return prompt.Confirm(message)
}

func handleRepair(args []string) {
	if len(args) > 1 {
		dieUsage("Usage: ctx repair [--activity]")
	}
	msg := ""
	var err error
	switch {
	case len(args) == 0:
		msg, err = setup.Repair()
	case args[0] == "--activity":
		msg, err = setup.RepairActivityLog()
	default:
		dieUsage("Usage: ctx repair [--activity]")
	}
	if err != nil {
		die(err)
	}
	if msg == "" {
		msg = "No issues detected."
	}
	fmt.Println(msg)
}

func handleResume(args []string) {
	if len(args) != 0 {
		dieUsage("Usage: ctx resume")
	}

	result, err := setup.Resume()
	if err != nil {
		die(err)
	}

	fmt.Println("Continuum storage:", result.BasePath)
	if result.RepairMessage != "" {
		fmt.Println("Repair:", result.RepairMessage)
	}
	switch {
	case result.Sync != nil:
		fmt.Printf("Sync: ok (push=%d pull=%d)\n", result.Sync.PushCount, result.Sync.PullCount)
	case result.SyncWarning != "":
		fmt.Println("Sync:", result.SyncWarning)
	default:
		fmt.Println("Sync: skipped")
	}
	fmt.Printf("Unsynced: %d commit(s)\n", result.UnsyncedCount)

	if len(result.Projects) == 0 {
		fmt.Println("Projects: none")
		return
	}

	fmt.Println("Projects:")
	for _, project := range result.Projects {
		activeTasks, err := task.ListWithStatus(project, string(task.StatusActive))
		if err != nil {
			fmt.Printf("  - %s (%s)\n", project, "task list unavailable")
			continue
		}
		if len(activeTasks) == 0 {
			fmt.Printf("  - %s (0 active tasks)\n", project)
			continue
		}
		names := make([]string, 0, len(activeTasks))
		for _, item := range activeTasks {
			names = append(names, item.Name)
		}
		fmt.Printf("  - %s (%d active tasks): %s\n", project, len(activeTasks), strings.Join(names, ", "))
	}
}

func handleWatch(args []string) {
	project, interval, tui, err := parseWatchArgs(args)
	if err != nil {
		die(err)
	}
	if tui {
		err = task.WatchTUI(project, interval)
	} else {
		err = task.Watch(project, interval)
	}
	if err != nil {
		die(err)
	}
}

func handleSearch(args []string) {
	project, taskName, query, limit, since, err := parseSearchArgsFull(args)
	if err != nil {
		die(err)
	}

	results, err := search.Search(query, project, taskName, limit, since)
	if err != nil {
		die(err)
	}
	if len(results) == 0 {
		fmt.Println("No matches found.")
		return
	}

	for _, result := range results {
		fmt.Printf("%s/%s %s:%d [%s]\n", result.Project, result.Task, result.File, result.Line, result.Kind)
		fmt.Printf("  %s\n", result.Text)
	}
}

func handleArtifact(args []string) {
	if len(args) < 2 {
		dieUsage(artifactListUsage, artifactShowUsage)
	}

	switch args[0] {
	case "list":
		taskName, project, captureType := parseArtifactListArgs(args[1:])
		project = resolveProject(project)
		artifacts, err := task.ListArtifacts(taskName, project, captureType)
		if err != nil {
			die(err)
		}
		if len(artifacts) == 0 {
			fmt.Println("No artifacts found.")
			return
		}
		for _, artifact := range artifacts {
			fmt.Printf("%s [%s]\n", artifact.Name, artifact.Type)
		}
	case "show":
		taskName, project, filename := parseArtifactFileArgs(args[1:])
		project = resolveProject(project)
		content, err := task.ReadArtifact(taskName, project, filename)
		if err != nil {
			die(err)
		}
		fmt.Print(content)
	default:
		fmt.Fprintln(os.Stderr, "Unknown artifact subcommand:", args[0])
		dieUsage(artifactListUsage, artifactShowUsage)
	}
}

func handleResolve(args []string) {
	if len(args) < 2 {
		dieUsage(resolveUsage)
	}
	taskName, project, filename := parseArtifactFileArgs(args)
	project = resolveProject(project)
	if err := task.ResolveArtifact(taskName, project, filename); err != nil {
		die(err)
	}
	fmt.Printf("Artifact '%s' resolved for %s/%s.\n", filename, project, taskName)
}

func handleHistory(args []string) {
	project, taskName, limit, since, err := parseHistoryArgs(args)
	if err != nil {
		die(err)
	}
	if project == "" && taskName == "" {
		project = resolveProject("")
	}
	output, err := history.Render(project, taskName, limit, since)
	if err != nil {
		die(err)
	}
	if output == "" {
		fmt.Println("No history found.")
		return
	}
	fmt.Println(output)
}

func handleTimeline(args []string) {
	project, taskName, limit, since, err := parseHistoryArgs(args)
	if err != nil {
		die(err)
	}
	if project == "" && taskName == "" {
		project = resolveProject("")
	}
	output, err := history.RenderTimeline(project, taskName, limit, since)
	if err != nil {
		die(err)
	}
	if output == "" {
		fmt.Println("No timeline found.")
		return
	}
	fmt.Println(output)
}

func handleDiff(args []string) {
	project, taskName, fromName, toName, err := parseDiffArgs(args)
	if err != nil {
		die(err)
	}
	project = resolveProject(project)

	output, err := task.Diff(taskName, project, fromName, toName)
	if err != nil {
		die(err)
	}
	fmt.Print(output)
}

func handleAgent(args []string) {
	if len(args) < 1 {
		dieUsage(agentUsage...)
	}

	switch args[0] {
	case "install":
		projectName, force := parseAgentInstallArgs(args[1:])
		if projectName == "" {
			dieUsage(agentInstallUsage)
		}
		if err := agent.Install(projectName, force); err != nil {
			die(err)
		}
		fmt.Printf("Installed Continuum bootstrap (project: %s).\n", projectName)
	case "status":
		projectName, _, err := parseAgentProjectArgs(args[1:])
		if err != nil {
			die(err)
		}
		projectName = resolveAgentProject("status", projectName)
		checks, err := agent.Status(projectName)
		if err != nil {
			die(err)
		}
		printAgentStatus(checks)
	case "update":
		projectName, force, err := parseAgentProjectArgs(args[1:])
		if err != nil {
			die(err)
		}
		projectName = resolveAgentProject("update", projectName)
		msg, err := agent.Update(projectName, force)
		if err != nil {
			die(err)
		}
		fmt.Println(msg)
	case "remove":
		for _, arg := range args[1:] {
			if _, ok := parseFlag(arg, "--project="); ok {
				continue
			}
			dieUsage(agentRemoveUsage)
		}
		if err := agent.Remove(); err != nil {
			die(err)
		}
		fmt.Println("Removed Continuum bootstrap.")
	default:
		fmt.Fprintln(os.Stderr, "Unknown agent command.")
		dieUsage(agentUsage...)
	}
}

func printAgentStatus(checks []agent.BootstrapCheck) {
	for _, check := range checks {
		installed := check.InstalledVersion
		if installed == "" {
			installed = "unknown"
		}
		current := check.CurrentVersion
		if current == "" {
			current = "unknown"
		}
		switch check.Status {
		case "ok":
			fmt.Printf("%s: installed bootstrap %s, current %s (ok)\n", check.File, installed, current)
		case "stale":
			fmt.Printf("%s: installed bootstrap %s, current %s (stale)\n", check.File, installed, current)
		case "missing":
			fmt.Printf("%s: missing (%s)\n", check.File, check.Detail)
		default:
			fmt.Printf("%s: unknown (%s)\n", check.File, check.Detail)
		}
	}
}

func handleExport(args []string) {
	if len(args) < 1 {
		dieUsage("Usage: ctx export [<task> | --project=<name[,name2...]> | --session] [--path=<destination>] [--encrypt[=<algo>]]",
			"  Supported algorithms: aes-gcm-v2")
	}

	projects, taskName, customPath, encryptAlgo, session, err := parseExportArgs(args)
	if err != nil {
		die(err)
	}
	if taskName != "" && len(projects) == 0 {
		projects = []string{resolveProject("")}
	} else if taskName != "" && len(projects) == 1 {
		projects[0] = resolveProject(projects[0])
	}

	var outputPath string
	if encryptAlgo != "" {
		if !encryptAlgo.Valid() {
			die(fmt.Errorf("invalid algorithm: %s", encryptAlgo))
		}
		switch {
		case session:
			outputPath, err = export.ExportSessionEncrypted(customPath, encryptAlgo)
		case taskName != "":
			outputPath, err = export.ExportTaskEncrypted(taskName, projects[0], customPath, encryptAlgo)
		default:
			outputPath, err = export.ExportProjectsEncrypted(projects, customPath, encryptAlgo)
		}
	} else {
		switch {
		case session:
			outputPath, err = export.ExportSession(customPath)
		case taskName != "":
			outputPath, err = export.ExportTask(taskName, projects[0], customPath)
		default:
			outputPath, err = export.ExportProjects(projects, customPath)
		}
	}
	if err != nil {
		die(err)
	}
	fmt.Println("Export written to:", outputPath)
}

func handleImport(args []string) {
	if len(args) < 1 {
		dieUsage("Usage: ctx import <zip-path> [--decrypt[=<algo>]]",
			"  Example: ctx import task.zip",
			"  Example: ctx import task.zip.enc --decrypt",
			"  Supported algorithms for decrypt: aes-gcm-v2")
	}

	zipPath, decrypt, algo := parseImportArgs(args)
	if decrypt && !algo.Valid() {
		die(fmt.Errorf("invalid algorithm: %s", algo))
	}

	target, err := export.ImportArchive(zipPath, decrypt, algo)
	if err != nil {
		die(err)
	}
	fmt.Println("Import completed:", target)
}

func handleHandoff(args []string) {
	if len(args) < 1 {
		dieUsage(handoffUsage)
	}
	taskName, project, autoConfirm := parseTaskArgs(args)
	project = resolveProject(project)
	if err := task.Handoff(taskName, project, autoConfirm); err != nil {
		die(err)
	}
}

func handleList(args []string) {
	project, statusFilter, err := parseListArgs(args)
	if err != nil {
		die(err)
	}
	project = resolveProject(project)
	tasks, err := task.ListWithStatus(project, statusFilter)
	if err != nil {
		die(err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}

	fmt.Printf("Tasks for project '%s':\n", project)
	for _, t := range tasks {
		if statusFilter == "" || statusFilter == string(task.StatusActive) {
			fmt.Printf("- %s\n", t.Name)
			continue
		}
		fmt.Printf("- %s (%s)\n", t.Name, t.Status)
	}
}

func handleTask(args []string) {
	if len(args) < 2 {
		dieUsage(taskUsage...)
	}

	subcommand := args[0]
	project, taskName := parseTaskCommandArgs(args[1:])
	project = resolveProject(project)

	switch subcommand {
	case "start":
		result, err := task.Start(taskName, project)
		if err != nil {
			die(err)
		}
		if result == task.StartAlreadyActive {
			fmt.Printf("Task '%s' is already active in project '%s'; no changes made.\n", taskName, project)
		} else {
			fmt.Printf("Task '%s' initialized in project '%s'.\n", taskName, project)
		}
	case "close":
		changed, err := task.SetStatus(taskName, project, task.StatusClosed)
		if err != nil {
			die(err)
		}
		if changed {
			fmt.Printf("Task '%s' closed in project '%s'.\n", taskName, project)
		} else {
			fmt.Printf("Task '%s' is already closed in project '%s'.\n", taskName, project)
		}
	case "reopen":
		changed, err := task.SetStatus(taskName, project, task.StatusActive)
		if err != nil {
			die(err)
		}
		if changed {
			fmt.Printf("Task '%s' reopened in project '%s'.\n", taskName, project)
		} else {
			fmt.Printf("Task '%s' is already active in project '%s'.\n", taskName, project)
		}
	case "delete":
		if err := task.DeleteTask(taskName, project); err != nil {
			die(err)
		}
		fmt.Printf("Task '%s' removed from project '%s'.\n", taskName, project)
	default:
		fmt.Fprintln(os.Stderr, "Unknown task subcommand:", subcommand)
		dieUsage(taskUsage...)
	}
}

func handleProject(args []string) {
	if len(args) < 1 {
		dieUsage(projectUsage...)
	}

	subcommand := args[0]

	switch subcommand {
	case "list":
		if len(args) != 1 {
			dieUsage("Usage: ctx project list")
		}
		handleProjectList()
	case "init":
		project, err := parseProjectCommandArgs(args[1:])
		if err != nil {
			dieUsage("Usage: ctx project init <project>")
		}
		if err := setup.Init(project, false); err != nil {
			die(err)
		}
		fmt.Printf("Continuum initialized for project '%s'.\n", project)
		fmt.Println("Templates: .ctx/templates/")
		fmt.Println("Edit these files to customize defaults.")
	case "onboard":
		project, force, autoConfirm, err := parseProjectOnboardArgs(args[1:])
		if err != nil {
			dieUsage(projectOnboardUsage)
		}
		content, err := readProjectOnboardContent()
		if err != nil {
			die(err)
		}
		if err := confirmProjectOnboard(project, string(content), autoConfirm, func() error {
			return setup.OnboardProject(project, content, force)
		}); err != nil {
			die(err)
		}
	case "delete":
		project, err := parseProjectCommandArgs(args[1:])
		if err != nil {
			dieUsage("Usage: ctx project delete <project>")
		}
		if err := setup.DeleteProject(project); err != nil {
			die(err)
		}
		fmt.Printf("Project '%s' removed.\n", project)
	default:
		fmt.Fprintln(os.Stderr, "Unknown project subcommand:", subcommand)
		dieUsage(projectUsage...)
	}
}

func readProjectOnboardContent() ([]byte, error) {
	if prompt.IsInteractiveInput() {
		return nil, fmt.Errorf("ctx project onboard expects markdown on stdin")
	}
	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("cannot read onboarding content: %w", err)
	}
	return content, nil
}

func confirmProjectOnboard(project, content string, autoConfirm bool, save func() error) error {
	fmt.Printf("\nProposed project context for '%s':\n\n%s\n\n", project, strings.TrimSpace(content))

	if autoConfirm {
		if err := save(); err != nil {
			return err
		}
		fmt.Println("Auto-confirmed with --yes.")
		fmt.Println("Project context saved.")
		return nil
	}

	ok, err := prompt.Confirm("Apply this project onboarding? [y] yes  [n] no\n> ")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("Discarded.")
		return nil
	}
	if err := save(); err != nil {
		return err
	}
	fmt.Println("Project context saved.")
	return nil
}

func handleSnapshot(args []string) {
	if len(args) < 2 {
		dieUsage(snapshotUsage)
	}

	subcommand := args[0]
	project, taskName, autoConfirm, keep, err := parseSnapshotArgs(args[1:])
	if err != nil {
		die(err)
	}
	project = resolveProject(project)

	switch subcommand {
	case "refresh":
		if err := task.SnapshotRefresh(taskName, project, autoConfirm); err != nil {
			die(err)
		}
	case "clean":
		if taskName == "" {
			dieUsage("Usage: ctx snapshot clean <task>")
		}
		removed, err := task.SnapshotClean(taskName, project, keep)
		if err != nil {
			die(err)
		}
		if removed == 0 {
			fmt.Println("No snapshots to remove.")
		} else {
			fmt.Printf("Snapshots cleaned: %d. Kept latest %d.\n", removed, keep)
		}
	default:
		dieUsage("Usage: ctx snapshot refresh <task>")
	}
}

func skillsBasePath() string {
	return setup.ResolvePath("skills")
}

var skillUsage = []string{
	"Usage: ctx skill <command> [options]",
	"Commands: list, show, save, delete",
	"  ctx skill list [--json]",
	"  ctx skill show <name> [--json]",
	"  ctx skill save <name> [--description=<text>] [--yes]",
	"  ctx skill delete <name> [--yes]",
}

func handleSkill(args []string) {
	if len(args) < 1 || args[0] == "--help" || args[0] == "-h" {
		dieUsage(skillUsage...)
	}

	switch args[0] {
	case "list":
		useJSON := false
		for _, arg := range args[1:] {
			if arg == "--json" {
				useJSON = true
			} else {
				dieUsage("Usage: ctx skill list [--json]")
			}
		}

		entries, fromIndex, err := skill.ListWithDescriptions(skillsBasePath())
		if err != nil {
			die(err)
		}
		if len(entries) == 0 {
			if useJSON {
				fmt.Println("[]")
			} else {
				fmt.Println("No skills found. Use ctx skill save <name> to create one.")
			}
			return
		}

		if useJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(entries)
			return
		}

		if fromIndex {
			maxLen := 0
			for _, e := range entries {
				if len(e.Name) > maxLen {
					maxLen = len(e.Name)
				}
			}
			for _, e := range entries {
				if e.Description != "" {
					fmt.Printf("%-*s  %s\n", maxLen, e.Name, e.Description)
				} else {
					fmt.Println(e.Name)
				}
			}
		} else {
			for _, e := range entries {
				fmt.Println(e.Name)
			}
		}

	case "show":
		if len(args) < 2 {
			dieUsage("Usage: ctx skill show <name>")
		}
		if strings.HasPrefix(args[1], "--") {
			dieUsage("Usage: ctx skill show <name>")
		}
		name := args[1]
		useJSON := false
		for _, arg := range args[2:] {
			if arg == "--json" {
				useJSON = true
			} else {
				dieUsage("Usage: ctx skill show <name> [--json]")
			}
		}

		content, err := skill.Show(skillsBasePath(), name)
		if err != nil {
			die(err)
		}

		if useJSON {
			json.NewEncoder(os.Stdout).Encode(struct {
				Name    string `json:"name"`
				Content string `json:"content"`
			}{name, content})
			return
		}
		fmt.Print(content)

	case "save":
		if len(args) < 2 {
			dieUsage("Usage: ctx skill save <name> [--description=<text>] [--yes]")
		}
		if strings.HasPrefix(args[1], "--") {
			dieUsage("Usage: ctx skill save <name> [--description=<text>] [--yes]")
		}
		name := args[1]
		autoConfirm := false
		description := ""
		for _, arg := range args[2:] {
			if arg == "--yes" {
				autoConfirm = true
			} else if val, ok := parseFlag(arg, "--description="); ok {
				description = val
			} else {
				dieUsage("Usage: ctx skill save <name> [--description=<text>] [--yes]")
			}
		}

		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			die(fmt.Errorf("cannot read skill content: %w", err))
		}
		if len(strings.TrimSpace(string(content))) == 0 {
			die(fmt.Errorf("skill content cannot be empty"))
		}

		base := skillsBasePath()

		if migrated, err := skill.MigrateAgentToIndex(base); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skills index migration failed: %v\n", err)
		} else if migrated {
			fmt.Printf("Created skills index at %s/index.md\n", base)
		}

		existing, showErr := skill.Show(base, name)
		if showErr == nil && !autoConfirm {
			fmt.Printf("Skill %q already exists:\n\n%s\n\n", name, existing)
			ok, err := prompt.Confirm("Overwrite? [y/N]: ")
			if err != nil {
				die(err)
			}
			if !ok {
				fmt.Println("Discarded.")
				return
			}
		}

		if err := skill.Save(base, name, string(content), autoConfirm || showErr != nil); err != nil {
			die(err)
		}
		if err := skill.UpdateIndex(base, name, description); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not update skills index: %v\n", err)
		}
		fmt.Printf("Skill %q saved.\n", name)

	case "delete":
		if len(args) < 2 {
			dieUsage("Usage: ctx skill delete <name> [--yes]")
		}
		if strings.HasPrefix(args[1], "--") {
			dieUsage("Usage: ctx skill delete <name> [--yes]")
		}
		name := args[1]
		autoConfirm := false
		for _, arg := range args[2:] {
			if arg == "--yes" {
				autoConfirm = true
			} else {
				dieUsage("Usage: ctx skill delete <name> [--yes]")
			}
		}

		if !autoConfirm {
			ok, err := prompt.Confirm(fmt.Sprintf("Delete skill %q? [y/N]: ", name))
			if err != nil {
				die(err)
			}
			if !ok {
				fmt.Println("Aborted.")
				return
			}
		}

		if err := skill.Delete(skillsBasePath(), name); err != nil {
			die(err)
		}
		fmt.Printf("Skill %q deleted.\n", name)

	default:
		fmt.Fprintln(os.Stderr, "Unknown skill subcommand:", args[0])
		dieUsage(skillUsage...)
	}
}

func handleConfig(args []string) {
	if len(args) < 1 {
		dieUsage(configSetUsage)
	}
	switch args[0] {
	case "set":
		key, value, err := parseConfigSetArgs(args[1:])
		if err != nil {
			die(err)
		}
		switch key {
		case "host":
			if err := identity.SetHost(value); err != nil {
				die(err)
			}
			fmt.Printf("Continuum host set to %q.\n", value)
		default:
			dieUsage(configSetUsage)
		}
	default:
		dieUsage(configSetUsage)
	}
}

func dispatchCommand(command string, args []string) {
	switch command {
	case "init":
		handleInit(args)
	case "capture":
		handleCapture(args)
	case "context":
		handleContext(args)
	case "sync":
		handleSync(args)
	case "resume":
		handleResume(args)
	case "repair":
		handleRepair(args)
	case "watch":
		handleWatch(args)
	case "search":
		handleSearch(args)
	case "artifact":
		handleArtifact(args)
	case "resolve":
		handleResolve(args)
	case "history":
		handleHistory(args)
	case "timeline":
		handleTimeline(args)
	case "diff":
		handleDiff(args)
	case "agent":
		handleAgent(args)
	case "export":
		handleExport(args)
	case "import":
		handleImport(args)
	case "handoff":
		handleHandoff(args)
	case "list":
		handleList(args)
	case "task":
		handleTask(args)
	case "project":
		handleProject(args)
	case "snapshot":
		handleSnapshot(args)
	case "skill":
		handleSkill(args)
	case "config":
		handleConfig(args)
	default:
		fmt.Fprintln(os.Stderr, "Unknown command:", command)
		printUsage()
		os.Exit(1) // printUsage already prints instructions
	}
}
