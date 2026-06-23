package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/mattn/go-isatty"
)

// ansiPattern matches ANSI escape sequences used for terminal styling.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func ConfirmReader(r io.Reader, message string) (bool, error) {
	reader := bufio.NewReader(r)
	fmt.Print(message)
	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("cannot read confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func Confirm(message string) (bool, error) {
	tty, err := os.Open("/dev/tty")
	if err == nil {
		defer tty.Close()
		return ConfirmReader(tty, message)
	}
	return ConfirmReader(os.Stdin, message)
}

// ReadLineReader reads a single line from r, trimming trailing newlines.
func ReadLineReader(r io.Reader) (string, error) {
	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("cannot read input: %w", err)
	}
	return strings.TrimRight(input, "\r\n"), nil
}

// ReadLine prints a prompt and reads a line from /dev/tty, falling back to
// os.Stdin if /dev/tty is unavailable.
func ReadLine(message string) (string, error) {
	fmt.Print(message)
	tty, err := os.Open("/dev/tty")
	if err == nil {
		defer tty.Close()
		return ReadLineReader(tty)
	}
	return ReadLineReader(os.Stdin)
}

// IsInteractiveOutput reports whether stdout is connected to a terminal.
func IsInteractiveOutput() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// AnsiStyle wraps ANSI escape codes into a formatted sequence.
func AnsiStyle(codes ...string) string {
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

// StripANSI removes ANSI escape sequences from a string.
func StripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}

// VisibleWidth returns the visible character width of a string, excluding
// ANSI escape sequences.
func VisibleWidth(value string) int {
	return len(StripANSI(value))
}

// IsInteractiveInput reports whether stdin is connected to a terminal.
func IsInteractiveInput() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}
