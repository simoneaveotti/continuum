package main

import (
	"fmt"
	"strings"

	"continuum/internal/context"
	"continuum/internal/prompt"
	"continuum/internal/setup"
	"continuum/internal/task"
	"continuum/internal/template"
)

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

	reporter := newProgressReporter()
	reporter.report("Syncing with remote...")
	result, err := setup.SyncWithOptions(setup.SyncOptions{
		Remote: remote,
		Prefer: prefer,
		Force:  force,
	})
	reporter.finish()
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
	reporter := newProgressReporter()
	switch {
	case len(args) == 0:
		reporter.report("Repairing storage...")
		msg, err = setup.Repair()
	case args[0] == "--activity":
		reporter.report("Repairing activity log...")
		msg, err = setup.RepairActivityLog()
	default:
		dieUsage("Usage: ctx repair [--activity]")
	}
	reporter.finish()
	if err != nil {
		die(err)
	}
	if msg == "" {
		msg = "No issues detected."
	}
	fmt.Println(msg)
}

func handleResume(args []string) {
	verbose, err := parseResumeArgs(args)
	if err != nil {
		dieUsage(err.Error())
	}

	reporter := newProgressReporter()
	result, err := setup.Resume(reporter.report)
	reporter.finish()
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

	projectCount := len(result.Projects)
	fmt.Printf("Projects: %d managed\n", projectCount)
	if !verbose || projectCount == 0 {
		return
	}

	fmt.Println("Project details:")
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
