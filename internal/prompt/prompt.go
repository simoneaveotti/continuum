package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

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
