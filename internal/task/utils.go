package task

import (
	"bufio"
	"fmt"
	"strings"

	"continuum/internal/parse"
	"continuum/internal/prompt"
)

func promptWithDefault(reader *bufio.Reader, label, current string) (string, error) {
	if current != "" {
		fmt.Printf("%s [%s]: ", label, current)
	} else {
		fmt.Printf("%s: ", label)
	}

	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	text = strings.TrimSpace(text)

	if text == "" {
		return current, nil
	}

	return text, nil
}

func parseSections(content string) map[string]string {
	return parse.Sections(content)
}

func cleanPrefill(value string) string {
	return parse.CleanValue(value)
}

func confirmAndSave(taskName, summary string, autoConfirm bool, save func() error, savedMessage string) error {
	fmt.Printf("\nProposed update for task '%s':\n\n%s\n\n", taskName, summary)

	if autoConfirm {
		if err := save(); err != nil {
			return err
		}
		fmt.Println("Auto-confirmed with --yes.")
		fmt.Println(savedMessage)
		return nil
	}

	ok, err := prompt.Confirm("Apply this update? [y] yes  [n] no\n> ")
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
	fmt.Println(savedMessage)
	return nil
}
