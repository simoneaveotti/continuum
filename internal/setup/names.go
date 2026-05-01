package setup

import (
	"fmt"
	"regexp"
)

var validNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

const maxNameLength = 64

func ValidateProjectName(name string) error {
	return validateName("project", name)
}

func ValidateTaskName(name string) error {
	return validateName("task", name)
}

func validateName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("%s name is required", kind)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid %s name %q", kind, name)
	}
	if len(name) > maxNameLength {
		return fmt.Errorf("invalid %s name %q: maximum length is %d characters", kind, name, maxNameLength)
	}
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("invalid %s name %q: use only letters, numbers, '.', '_' or '-', and start with a letter or number", kind, name)
	}
	return nil
}
