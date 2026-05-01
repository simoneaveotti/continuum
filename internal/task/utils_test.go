package task

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}

func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()

	original := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	os.Stdin = r
	defer func() { os.Stdin = original }()

	go func() {
		_, _ = w.WriteString(input)
		_ = w.Close()
	}()

	fn()
}

func TestConfirmAndSaveAutoConfirmSkipsPrompt(t *testing.T) {
	called := false

	output := captureStdout(t, func() {
		err := confirmAndSave("demo", "State: ok", true, true, func() error {
			called = true
			return nil
		}, "State saved.")
		if err != nil {
			t.Fatalf("confirmAndSave: %v", err)
		}
	})

	if !called {
		t.Fatal("expected save to be called")
	}
	if strings.Contains(output, "Apply this update?") {
		t.Fatalf("expected no interactive prompt in auto-confirm output, got %q", output)
	}
	if !strings.Contains(output, "Auto-confirmed with --yes.") {
		t.Fatalf("expected auto-confirm message, got %q", output)
	}
}

func TestConfirmAndSaveInteractiveShowsPrompt(t *testing.T) {
	output := captureStdout(t, func() {
		withStdin(t, "n\n", func() {
			_ = confirmAndSave("demo", "State: ok", false, false, func() error {
				return nil
			}, "State saved.")
		})
	})

	if !strings.Contains(output, "Apply this update?") {
		t.Fatalf("expected interactive prompt, got %q", output)
	}
}
