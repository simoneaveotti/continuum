package setup

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"continuum/internal/vcs"
)

func TestRemoteOperationError(t *testing.T) {
	for _, tc := range []struct {
		name, diagnostic, want string
	}{
		{"https authentication", "fatal: Authentication failed for 'https://user:SECRET@example.com/repo'", "Renew your token and reauthenticate"},
		{"github token", "remote: Invalid username or token.", "Renew your token and reauthenticate"},
		{"gitlab token", "remote: HTTP Basic: Access denied", "Renew your token and reauthenticate"},
		{"expired token", "remote: token has expired", "Renew your token and reauthenticate"},
		{"missing credentials", "fatal: could not read Username: terminal prompts disabled", "Renew your token and reauthenticate"},
		{"http unauthorized", "fatal: requested URL returned error: 401", "Renew your token and reauthenticate"},
		{"ssh", "git@example.com: Permission denied (publickey).", "SSH authentication failed"},
		{"network", "Could not resolve host: example.com", "Could not push to origin."},
		{"branch permissions", "remote: protected branch; requested URL returned error: 403", "Could not push to origin."},
		{"lease rejection", "! [rejected] main -> main (stale info)", "Could not push to origin."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cause := &vcs.GitError{Op: "push", Stderr: tc.diagnostic}
			err := remoteOperationError("push to", fmt.Errorf("wrapped: %w", cause))
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %q, want %q", err, tc.want)
			}
			if strings.Contains(err.Error(), "SECRET") || strings.Contains(err.Error(), "example.com") {
				t.Fatalf("raw diagnostic exposed: %q", err)
			}
			if tc.want == "Could not push to origin." && strings.Contains(err.Error(), "reauthenticate") {
				t.Fatalf("non-authentication failure classified as credentials: %q", err)
			}
			if !errors.Is(err, cause) {
				t.Fatal("underlying Git error was lost")
			}
		})
	}
}
