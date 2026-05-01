package parse

import (
	"testing"
)

func TestSections(t *testing.T) {
	content := `# Title

## Objective
Build something

## Current State
- step 1
- step 2

## Next Step
Deploy
`
	got := Sections(content)

	if got["Objective"] != "Build something" {
		t.Errorf("Objective = %q, want %q", got["Objective"], "Build something")
	}
	if got["Next Step"] != "Deploy" {
		t.Errorf("Next Step = %q, want %q", got["Next Step"], "Deploy")
	}
	if _, ok := got["Title"]; ok {
		t.Error("# heading should not be captured as a section")
	}
}

func TestSectionsEmpty(t *testing.T) {
	got := Sections("")
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestCleanValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple value", "simple value"},
		{"- list item", "list item"},
		{"- item one\n- item two", "item one | item two"},
		{"  spaced  ", "spaced"},
		{"", ""},
		{"   ", ""},
		{"-", ""},
		{"- \n- ", ""},
	}

	for _, tt := range tests {
		got := CleanValue(tt.input)
		if got != tt.want {
			t.Errorf("CleanValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
