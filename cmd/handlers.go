package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"continuum/internal/agent"
	"continuum/internal/export"
	"continuum/internal/identity"
	"continuum/internal/prompt"
	"continuum/internal/search"
	"continuum/internal/setup"
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
		results, err := agent.Install(projectName, force)
		if err != nil {
			die(err)
		}
		printAgentFileResults(results)
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
		results, err := agent.Update(projectName, force)
		if err != nil {
			die(err)
		}
		if results == nil {
			fmt.Println("Agent bootstrap already current.")
		} else {
			printAgentFileResults(results)
		}
	case "remove":
		for _, arg := range args[1:] {
			if _, ok := parseFlag(arg, "--project="); ok {
				continue
			}
			dieUsage(agentRemoveUsage)
		}
		results, err := agent.Remove()
		if err != nil {
			die(err)
		}
		printAgentRemoveResults(results)
	default:
		fmt.Fprintln(os.Stderr, "Unknown agent command.")
		dieUsage(agentUsage...)
	}
}

func printAgentFileResults(results []agent.FileResult) {
	installed, skipped, errored := 0, 0, 0
	for _, result := range results {
		switch result.Status {
		case "installed":
			installed++
			fmt.Printf("%s: installed\n", result.Filename)
		case "skipped":
			skipped++
			fmt.Printf("%s: skipped (%s)\n", result.Filename, result.Detail)
		case "error":
			errored++
			fmt.Printf("%s: error (%s)\n", result.Filename, result.Detail)
		}
	}
	fmt.Printf("Installed: %d, Skipped: %d, Errors: %d\n", installed, skipped, errored)
}

func printAgentRemoveResults(results []agent.FileResult) {
	removed, skipped, errored := 0, 0, 0
	for _, result := range results {
		switch result.Status {
		case "removed":
			removed++
			fmt.Printf("%s: bootstrap removed\n", result.Filename)
		case "skipped":
			skipped++
			fmt.Printf("%s: skipped (%s)\n", result.Filename, result.Detail)
		case "error":
			errored++
			fmt.Printf("%s: error (%s)\n", result.Filename, result.Detail)
		}
	}
	fmt.Printf("Removed: %d, Skipped: %d, Errors: %d\n", removed, skipped, errored)
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
		project, autoConfirm, err := parseProjectDeleteArgs(args[1:])
		if err != nil {
			dieUsage("Usage: ctx project delete <project> [--yes]")
		}
		if !autoConfirm {
			ok, err := prompt.Confirm(fmt.Sprintf("Delete project %q and all its tasks? [y/N]: ", project))
			if err != nil {
				die(err)
			}
			if !ok {
				fmt.Println("Delete canceled.")
				return
			}
		}
		reporter := newProgressReporter()
		reporter.report("Deleting project...")
		err = setup.DeleteProject(project)
		reporter.finish()
		if err != nil {
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
