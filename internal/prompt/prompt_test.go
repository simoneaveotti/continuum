package prompt

import (
	"errors"
	"strings"
	"testing"
)

func TestConfirmReaderAcceptsYes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"y lowercase", "y\n", true},
		{"Y uppercase", "Y\n", true},
		{"yes lowercase", "yes\n", true},
		{"YES uppercase", "YES\n", true},
		{"yes mixed", "Yes\n", true},
		{"n lowercase", "n\n", false},
		{"N uppercase", "N\n", false},
		{"no lowercase", "no\n", false},
		{"empty input", "\n", false},
		{"random text", "maybe\n", false},
		{"y prefix only", "yep\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConfirmReader(strings.NewReader(tt.input), "Continue? [y/N]: ")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ConfirmReader(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestConfirmReaderReturnsErrorOnReadFailure(t *testing.T) {
	_, err := ConfirmReader(&errReader{errors.New("read error")}, "test: ")
	if err == nil {
		t.Fatal("expected error from broken reader")
	}
}

func TestConfirmReaderDoesNotErrorOnEOF(t *testing.T) {
	got, err := ConfirmReader(strings.NewReader(""), "test: ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Errorf("expected false on empty input, got true")
	}
}

func TestReadLineReturnsInput(t *testing.T) {
	r := strings.NewReader("hello world\n")
	got, err := ReadLineReader(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("ReadLine = %q, want %q", got, "hello world")
	}
}

func TestReadLineTrimsCRLF(t *testing.T) {
	r := strings.NewReader("passphrase\r\n")
	got, err := ReadLineReader(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "passphrase" {
		t.Errorf("ReadLine = %q, want %q", got, "passphrase")
	}
}

func TestReadLineReturnsErrorOnReadFailure(t *testing.T) {
	_, err := ReadLineReader(&errReader{errors.New("read error")})
	if err == nil {
		t.Fatal("expected error from broken reader")
	}
}

type errReader struct{ err error }

func (e *errReader) Read([]byte) (int, error) { return 0, e.err }
