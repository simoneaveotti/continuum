package setup

import (
	"errors"
	"fmt"
	"strings"

	"continuum/internal/vcs"
)

// remoteOperationError classifies Git diagnostics without exposing remote URLs
// or credentials that may occur in stderr.
func remoteOperationError(operation string, err error) error {
	message := fmt.Sprintf("Could not %s origin.", operation)
	var gitErr *vcs.GitError
	if errors.As(err, &gitErr) {
		diagnostic := strings.ToLower(gitErr.Stderr)
		if strings.Contains(diagnostic, "permission denied (publickey)") {
			message += " SSH authentication failed. Check that your SSH key is loaded and authorized for the remote repository."
		} else {
			for _, marker := range []string{
				"authentication failed", "invalid username or token", "invalid username or password",
				"http basic: access denied", "could not read username", "could not read password",
				"returned error: 401", "token has expired", "token is expired",
			} {
				if strings.Contains(diagnostic, marker) {
					message += " Git authentication failed: your token may have expired or been revoked, or your saved credentials may be invalid. Renew your token and reauthenticate with your Git provider, then update Git's saved credentials and retry."
					break
				}
			}
		}
	}
	return fmt.Errorf("%s %w", message, err)
}
