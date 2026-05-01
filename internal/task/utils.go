package task

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"continuum/internal/parse"
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

func confirmAndSave(taskName, summary string, piped, autoConfirm bool, save func() error, savedMessage string) error {
	if autoConfirm {
		fmt.Printf("\nProposed update for task '%s':\n\n%s\n\n", taskName, summary)
		if err := save(); err != nil {
			return err
		}
		fmt.Println("Auto-confirmed with --yes.")
		fmt.Println(savedMessage)
		return nil
	}

	fmt.Printf("\nProposed update for task '%s':\n\n%s\n\nApply this update? [y] yes  [n] no\n",
		taskName, summary)

	fmt.Print("> ")

	var confirmReader *bufio.Reader
	if piped {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return fmt.Errorf("cannot open /dev/tty for confirmation: %w", err)
		}
		defer tty.Close()
		confirmReader = bufio.NewReader(tty)
	} else {
		confirmReader = bufio.NewReader(os.Stdin)
	}

	input, err := confirmReader.ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("cannot read input: %w", err)
	}

	switch strings.TrimSpace(strings.ToLower(input)) {
	case "y", "yes":
		if err := save(); err != nil {
			return err
		}
		fmt.Println(savedMessage)
	default:
		fmt.Println("Discarded.")
	}

	return nil
}
