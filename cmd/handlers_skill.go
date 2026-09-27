package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"continuum/internal/prompt"
	"continuum/internal/setup"
	"continuum/internal/skill"
)

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
			for _, entry := range entries {
				if len(entry.Name) > maxLen {
					maxLen = len(entry.Name)
				}
			}
			for _, entry := range entries {
				if entry.Description != "" {
					fmt.Printf("%-*s  %s\n", maxLen, entry.Name, entry.Description)
				} else {
					fmt.Println(entry.Name)
				}
			}
			return
		}
		for _, entry := range entries {
			fmt.Println(entry.Name)
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
			} else if value, ok := parseFlag(arg, "--description="); ok {
				description = value
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
