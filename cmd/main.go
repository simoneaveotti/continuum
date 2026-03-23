package main

import (
	"fmt"
	"os"

	"continuum/internal/context"
	"continuum/internal/export"
	"continuum/internal/render"
	"continuum/internal/setup"
	"continuum/internal/task"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "init":
		if err := setup.Init(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

	case "resume":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ctx resume <task> [--prompt-only]")
			os.Exit(1)
		}

		taskName := os.Args[2]
		promptOnly := len(os.Args) >= 4 && os.Args[3] == "--prompt-only"

		ctxData, err := context.Load(taskName)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		if promptOnly {
			render.PrintPromptOnly(ctxData)
		} else {
			render.PrintFull(ctxData)
		}

	case "export":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ctx export <task>")
			os.Exit(1)
		}

		taskName := os.Args[2]
		outputPath, err := export.LoadAndExport(taskName)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		fmt.Println("Export written to:", outputPath)

	case "handoff":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ctx handoff <task>")
			os.Exit(1)
		}

		taskName := os.Args[2]
		if err := task.Handoff(taskName); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

	case "list":
		tasks, err := task.List()
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return
		}

		fmt.Println("Tasks:")
		for _, t := range tasks {
			fmt.Printf("- %s\n", t)
		}

	case "task":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ctx task start <task>")
			os.Exit(1)
		}

		subcommand := os.Args[2]
		taskName := os.Args[3]

		switch subcommand {
		case "start":
			if err := task.Start(taskName); err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
		default:
			fmt.Println("Unknown task subcommand:", subcommand)
			fmt.Println("Usage: ctx task start <task>")
			os.Exit(1)
		}

	case "snapshot":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ctx snapshot refresh <task>")
			os.Exit(1)
		}

		subcommand := os.Args[2]
		taskName := os.Args[3]

		switch subcommand {
		case "refresh":
			if err := task.SnapshotRefresh(taskName); err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
		default:
			fmt.Println("Usage: ctx snapshot refresh <task>")
			os.Exit(1)
		}

	default:
		fmt.Println("Unknown command:", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  ctx init")
	fmt.Println("  ctx list")
	fmt.Println("  ctx resume <task> [--prompt-only]")
	fmt.Println("  ctx export <task>")
	fmt.Println("  ctx handoff <task>")
	fmt.Println("  ctx task start <task>")
	fmt.Println("  ctx snapshot refresh <task>")
}
