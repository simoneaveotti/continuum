package main

import (
	"fmt"
	"os"

	"continuum/internal/filestore"
	"continuum/internal/history"
	"continuum/internal/task"
)

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
