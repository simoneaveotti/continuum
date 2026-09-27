package main

import (
	"fmt"
	"os"

	"continuum/internal/prompt"
	"continuum/internal/task"
)

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
	project, taskName, autoConfirm := parseTaskCommandArgs(args[1:])
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
		if !autoConfirm {
			ok, err := prompt.Confirm(fmt.Sprintf("Delete task %q from project %q? [y/N]: ", taskName, project))
			if err != nil {
				die(err)
			}
			if !ok {
				fmt.Println("Delete canceled.")
				return
			}
		}
		reporter := newProgressReporter()
		reporter.report("Removing task...")
		err := task.DeleteTask(taskName, project)
		reporter.finish()
		if err != nil {
			die(err)
		}
		fmt.Printf("Task '%s' removed from project '%s'.\n", taskName, project)
	default:
		fmt.Fprintln(os.Stderr, "Unknown task subcommand:", subcommand)
		dieUsage(taskUsage...)
	}
}
