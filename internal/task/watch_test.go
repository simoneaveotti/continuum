package task

import "testing"

func TestJoinSummaryParts_SkipsEmptyAndPlaceholder(t *testing.T) {
	project, task := splitWatchKey("alpha/my-task")
	if project != "alpha" || task != "my-task" {
		t.Fatalf("splitWatchKey() = %q, %q", project, task)
	}
}

func TestWatchProjectsAllReturnsNoStaticFilter(t *testing.T) {
	projects, err := watchProjects("")
	if err != nil {
		t.Fatalf("watchProjects(all): %v", err)
	}
	if projects != nil {
		t.Fatalf("expected nil project filter for all-project watch, got %+v", projects)
	}
}
