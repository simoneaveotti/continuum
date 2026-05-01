package main

import (
	"bufio"
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
	"continuum/internal/search"
	"continuum/internal/setup"
	"continuum/internal/task"
	"continuum/internal/template"

	"github.com/mattn/go-isatty"
)

// die prints an error to stderr and exits with code 1.
func die(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
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

func handleInit(args []string) {
	projectName, templatesPath, force, remote := parseInitArgs(args)
	if templatesPath != "" {
		if err := template.ValidateSourcePath(templatesPath); err != nil {
			die(err)
		}
		template.SetSourcePath(templatesPath)
	}
	if projectName != "" {
		die(fmt.Errorf("Usage: ctx init [--templates=<path>] [--remote=<url>] [--force]"))
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
		fmt.Fprintln(os.Stderr, "Usage: ctx capture <task> --project=<name> [--type=state|proposal|request|response|decision] [--resolves=<filename>] [--yes]")
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
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
	if !isatty.IsTerminal(os.Stdout.Fd()) || (!isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd())) {
		return false, fmt.Errorf("non-interactive sync requires --force with --prefer=%s", prefer)
	}

	message := "This will preserve local uncommitted Continuum changes and sync them to the remote. Continue? [y/N]: "
	if prefer == "remote" {
		message = "This will discard local uncommitted Continuum changes and resync from the remote. Continue? [y/N]: "
	}

	var reader *bufio.Reader
	tty, err := os.Open("/dev/tty")
	if err == nil {
		defer tty.Close()
		reader = bufio.NewReader(tty)
	} else {
		reader = bufio.NewReader(os.Stdin)
	}

	fmt.Print(message)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("cannot read sync confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func handleRepair(args []string) {
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "Usage: ctx repair [--activity]")
		os.Exit(1)
	}
	msg := ""
	var err error
	switch {
	case len(args) == 0:
		msg, err = setup.Repair()
	case args[0] == "--activity":
		msg, err = setup.RepairActivityLog()
	default:
		fmt.Fprintln(os.Stderr, "Usage: ctx repair [--activity]")
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "Usage: ctx resume")
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "Usage: ctx artifact list <task> [--project=<name>] [--type=proposal|request|response|decision|all]")
		fmt.Fprintln(os.Stderr, "       ctx artifact show <task> <filename> [--project=<name>]")
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "Usage: ctx artifact list|show <task> [filename] [--project=<name>]")
		os.Exit(1)
	}
}

func handleResolve(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ctx resolve <task> <filename> [--project=<name>]")
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "Usage: ctx agent install --project=<name> [--force]")
		fmt.Fprintln(os.Stderr, "       ctx agent remove [--project=<name>]")
		os.Exit(1)
	}

	switch args[0] {
	case "install":
		projectName, force := parseAgentInstallArgs(args[1:])
		if projectName == "" {
			fmt.Fprintln(os.Stderr, "Usage: ctx agent install --project=<name> [--force]")
			os.Exit(1)
		}
		if err := agent.Install(projectName, force); err != nil {
			die(err)
		}
	case "remove":
		for _, arg := range args[1:] {
			if _, ok := parseFlag(arg, "--project="); ok {
				continue
			}
			fmt.Fprintln(os.Stderr, "Usage: ctx agent remove [--project=<name>]")
			os.Exit(1)
		}
		if err := agent.Remove(); err != nil {
			die(err)
		}
	default:
		fmt.Fprintln(os.Stderr, "Unknown agent command.")
		fmt.Fprintln(os.Stderr, "Usage: ctx agent install --project=<name> [--force]")
		fmt.Fprintln(os.Stderr, "       ctx agent remove [--project=<name>]")
		os.Exit(1)
	}
}

func handleExport(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ctx export [<task> | --project=<name[,name2...]> | --session] [--path=<destination>] [--encrypt[=<algo>]]")
		fmt.Fprintln(os.Stderr, "  Supported algorithms: aes-gcm-v2")
		os.Exit(1)
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
			fmt.Fprintln(os.Stderr, "Error: invalid algorithm:", encryptAlgo)
			os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "Usage: ctx import <zip-path> [--decrypt[=<algo>]]")
		fmt.Fprintln(os.Stderr, "  Example: ctx import task.zip")
		fmt.Fprintln(os.Stderr, "  Example: ctx import task.zip.enc --decrypt")
		fmt.Fprintln(os.Stderr, "  Supported algorithms for decrypt: aes-gcm-v2")
		os.Exit(1)
	}

	zipPath, decrypt, algo := parseImportArgs(args)
	if decrypt && !algo.Valid() {
		fmt.Fprintln(os.Stderr, "Error: invalid algorithm:", algo)
		os.Exit(1)
	}

	target, err := export.ImportArchive(zipPath, decrypt, algo)
	if err != nil {
		die(err)
	}
	fmt.Println("Import completed:", target)
}

func handleHandoff(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ctx handoff <task> [--project=<name>] [--yes]")
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "Usage: ctx task start <task> [--project=<name>]")
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "Usage: ctx task start|close|reopen|delete <task> [--project=<name>]")
		os.Exit(1)
	}
}

func handleProject(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ctx project list")
		fmt.Fprintln(os.Stderr, "       ctx project init <project>")
		fmt.Fprintln(os.Stderr, "       ctx project onboard <project>")
		fmt.Fprintln(os.Stderr, "       ctx project delete <project>")
		os.Exit(1)
	}

	subcommand := args[0]

	switch subcommand {
	case "list":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "Usage: ctx project list")
			os.Exit(1)
		}
		handleProjectList()
	case "init":
		project, err := parseProjectCommandArgs(args[1:])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Usage: ctx project init <project>")
			os.Exit(1)
		}
		if err := setup.Init(project, false); err != nil {
			die(err)
		}
	case "onboard":
		project, force, autoConfirm, err := parseProjectOnboardArgs(args[1:])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Usage: ctx project onboard <project> [--force] [--yes]")
			os.Exit(1)
		}
		content, piped, err := readProjectOnboardContent()
		if err != nil {
			die(err)
		}
		if err := confirmProjectOnboard(project, string(content), piped, autoConfirm, func() error {
			return setup.OnboardProject(project, content, force)
		}); err != nil {
			die(err)
		}
	case "delete":
		project, err := parseProjectCommandArgs(args[1:])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Usage: ctx project delete <project>")
			os.Exit(1)
		}
		if err := setup.DeleteProject(project); err != nil {
			die(err)
		}
		fmt.Printf("Project '%s' removed.\n", project)
	default:
		fmt.Fprintln(os.Stderr, "Unknown project subcommand:", subcommand)
		fmt.Fprintln(os.Stderr, "Usage: ctx project list|init|onboard|delete ...")
		os.Exit(1)
	}
}

func readProjectOnboardContent() ([]byte, bool, error) {
	piped := !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd())
	if !piped {
		return nil, false, fmt.Errorf("ctx project onboard expects markdown on stdin")
	}
	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, true, fmt.Errorf("cannot read onboarding content: %w", err)
	}
	return content, true, nil
}

func confirmProjectOnboard(project, content string, piped, autoConfirm bool, save func() error) error {
	if autoConfirm {
		fmt.Printf("\nProposed project context for '%s':\n\n%s\n\n", project, strings.TrimSpace(content))
		if err := save(); err != nil {
			return err
		}
		fmt.Println("Auto-confirmed with --yes.")
		fmt.Println("Project context saved.")
		return nil
	}

	fmt.Printf("\nProposed project context for '%s':\n\n%s\n\nApply this project onboarding? [y] yes  [n] no\n", project, strings.TrimSpace(content))
	fmt.Print("> ")

	var reader *bufio.Reader
	if piped {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return fmt.Errorf("cannot open /dev/tty for confirmation: %w", err)
		}
		defer tty.Close()
		reader = bufio.NewReader(tty)
	} else {
		reader = bufio.NewReader(os.Stdin)
	}

	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("cannot read input: %w", err)
	}
	switch strings.TrimSpace(strings.ToLower(input)) {
	case "y", "yes":
		if err := save(); err != nil {
			return err
		}
		fmt.Println("Project context saved.")
	default:
		fmt.Println("Discarded.")
	}
	return nil
}

func handleSnapshot(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ctx snapshot refresh|clean <task> [--project=<name>] [--yes] [--keep=N]")
		os.Exit(1)
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
			fmt.Fprintln(os.Stderr, "Usage: ctx snapshot clean <task>")
			os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "Usage: ctx snapshot refresh <task>")
		os.Exit(1)
	}
}

func handleConfig(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ctx config set host <name>")
		os.Exit(1)
	}
	switch args[0] {
	case "set":
		key, value, err := parseConfigSetArgs(args[1:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		switch key {
		case "host":
			if err := identity.SetHost(value); err != nil {
				die(err)
			}
			fmt.Printf("Continuum host set to %q.\n", value)
		default:
			fmt.Fprintln(os.Stderr, "Usage: ctx config set host <name>")
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "Usage: ctx config set host <name>")
		os.Exit(1)
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
	case "config":
		handleConfig(args)
	default:
		fmt.Fprintln(os.Stderr, "Unknown command:", command)
		printUsage()
		os.Exit(1)
	}
}
