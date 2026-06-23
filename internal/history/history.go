package history

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"continuum/internal/events"
	"continuum/internal/filestore"
	"continuum/internal/parse"
	"continuum/internal/prompt"
	"continuum/internal/setup"
)

type milestone struct {
	Timestamp string
	Project   string
	Task      string
	Kind      string
	Title     string
	State     string
	Next      string
	Source    string
}

var (
	historyHeaderStyle = ansiStyle("1", "38;5;110")
	historyIndexStyle  = ansiStyle("1", "38;5;73")
	historyTargetStyle = ansiStyle("38;5;221")
	historyLabelStyle  = ansiStyle("38;5;245")
	historyResetStyle  = "\x1b[0m"
)

func Render(projectFilter, taskFilter string, limit int, since time.Duration) (string, error) {
	if err := validateFilters(projectFilter, taskFilter); err != nil {
		return "", err
	}
	items, err := collectMilestones(projectFilter, taskFilter, since)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", nil
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Timestamp < items[j].Timestamp
	})
	if limit > 0 && len(items) > limit {
		items = items[len(items)-limit:]
	}

	var lines []string
	lines = append(lines, styleHeader(historyHeader("History", projectFilter, taskFilter, since, len(items), "milestone(s)")))
	for i, item := range items {
		lines = append(lines, renderMilestone(i+1, item)...)
	}
	return strings.Join(lines, "\n"), nil
}

func RenderTimeline(projectFilter, taskFilter string, limit int, since time.Duration) (string, error) {
	if err := validateFilters(projectFilter, taskFilter); err != nil {
		return "", err
	}
	items, _, err := events.ReadFromOffset(0)
	if err != nil {
		return "", err
	}
	items = filterTimelineEvents(items, projectFilter, taskFilter, since)
	if len(items) == 0 {
		return "", nil
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Timestamp < items[j].Timestamp
	})
	if limit > 0 && len(items) > limit {
		items = items[len(items)-limit:]
	}

	var lines []string
	lines = append(lines, styleHeader(historyHeader("Timeline", projectFilter, taskFilter, since, len(items), "event(s)")))
	for _, item := range items {
		lines = append(lines, renderTimelineEvent(item))
	}
	return strings.Join(lines, "\n"), nil
}

func validateFilters(projectFilter, taskFilter string) error {
	if projectFilter != "" {
		if err := setup.ValidateProjectName(projectFilter); err != nil {
			return err
		}
	}
	if taskFilter != "" {
		if err := setup.ValidateTaskName(taskFilter); err != nil {
			return err
		}
	}
	return nil
}

func collectMilestones(projectFilter, taskFilter string, since time.Duration) ([]milestone, error) {
	projects, err := resolveProjects(projectFilter)
	if err != nil {
		return nil, err
	}
	base := setup.ContinuumPath()
	var items []milestone
	for _, project := range projects {
		tasksDir := filepath.Join(base, "projects", project, "tasks")
		entries, err := os.ReadDir(tasksDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("cannot read tasks for project %q: %w", project, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			taskName := entry.Name()
			if taskFilter != "" && taskName != taskFilter {
				continue
			}
			taskDir := filepath.Join(tasksDir, taskName)
			taskItems, err := collectTaskMilestones(project, taskName, taskDir, since)
			if err != nil {
				return nil, err
			}
			items = append(items, taskItems...)
		}
	}
	return compressMilestones(items), nil
}

func collectTaskMilestones(project, taskName, taskDir string, since time.Duration) ([]milestone, error) {
	var items []milestone

	snapshots, err := filestore.AllSnapshots(taskDir)
	if err != nil {
		return nil, fmt.Errorf("cannot list snapshots for %s/%s: %w", project, taskName, err)
	}
	for _, path := range snapshots {
		item, ok, err := milestoneFromSnapshot(project, taskName, path, since)
		if err != nil {
			return nil, err
		}
		if ok {
			items = append(items, item)
		}
	}

	handoffs, err := filestore.AllHandoffs(taskDir)
	if err != nil {
		return nil, fmt.Errorf("cannot list handoffs for %s/%s: %w", project, taskName, err)
	}
	for _, path := range handoffs {
		item, ok, err := milestoneFromHandoff(project, taskName, path, since)
		if err != nil {
			return nil, err
		}
		if ok {
			items = append(items, item)
		}
	}

	return items, nil
}

func milestoneFromSnapshot(project, taskName, path string, since time.Duration) (milestone, bool, error) {
	name := filepath.Base(path)
	ts, ok := parseMilestoneTimestamp(name)
	if !ok {
		return milestone{}, false, nil
	}
	if since > 0 && time.Since(ts) > since {
		return milestone{}, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return milestone{}, false, fmt.Errorf("cannot read snapshot %s: %w", path, err)
	}
	content := string(data)
	title := firstNonEmpty(
		parse.ExtractField(content, "objective"),
		parse.ExtractField(content, "current state"),
		parse.ExtractField(content, "next step"),
		"snapshot updated",
	)
	state := extractState(content)
	next := parse.ExtractField(content, "next step")
	return milestone{
		Timestamp: ts.UTC().Format(time.RFC3339),
		Project:   project,
		Task:      taskName,
		Kind:      "snapshot",
		Title:     title,
		State:     state,
		Next:      next,
		Source:    name,
	}, true, nil
}

func milestoneFromHandoff(project, taskName, path string, since time.Duration) (milestone, bool, error) {
	name := filepath.Base(path)
	ts, ok := parseMilestoneTimestamp(name)
	if !ok {
		return milestone{}, false, nil
	}
	if since > 0 && time.Since(ts) > since {
		return milestone{}, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return milestone{}, false, fmt.Errorf("cannot read handoff %s: %w", path, err)
	}
	content := string(data)
	title := firstNonEmpty(
		parse.ExtractField(content, "objective"),
		parse.ExtractField(content, "what was done"),
		parse.ExtractField(content, "current state"),
		"handoff saved",
	)
	state := firstNonEmpty(
		parse.ExtractField(content, "what was done"),
		parse.ExtractField(content, "current state"),
	)
	next := firstNonEmpty(
		parse.ExtractField(content, "next recommended step"),
		parse.ExtractField(content, "next step"),
	)
	return milestone{
		Timestamp: ts.UTC().Format(time.RFC3339),
		Project:   project,
		Task:      taskName,
		Kind:      "handoff",
		Title:     title,
		State:     state,
		Next:      next,
		Source:    name,
	}, true, nil
}

func compressMilestones(items []milestone) []milestone {
	if len(items) == 0 {
		return nil
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Timestamp < items[j].Timestamp
	})
	var compressed []milestone
	for _, item := range items {
		n := len(compressed)
		if n > 0 {
			prev := compressed[n-1]
			if prev.Project == item.Project &&
				prev.Task == item.Task &&
				prev.Kind == item.Kind &&
				prev.Title == item.Title &&
				prev.State == item.State &&
				prev.Next == item.Next {
				continue
			}
		}
		compressed = append(compressed, item)
	}
	return compressed
}

func resolveProjects(projectFilter string) ([]string, error) {
	if projectFilter != "" {
		return []string{projectFilter}, nil
	}
	return setup.ListProjects()
}

func parseMilestoneTimestamp(name string) (time.Time, bool) {
	parts := strings.Split(name, ".")
	if len(parts) < 4 {
		return time.Time{}, false
	}
	ts, err := time.Parse("20060102T150405Z", parts[1])
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

func historyHeader(kind, projectFilter, taskFilter string, since time.Duration, count int, label string) string {
	scope := "all projects"
	switch {
	case projectFilter != "" && taskFilter != "":
		scope = fmt.Sprintf("project %q, task %q", projectFilter, taskFilter)
	case projectFilter != "":
		scope = fmt.Sprintf("project %q", projectFilter)
	case taskFilter != "":
		scope = fmt.Sprintf("task %q", taskFilter)
	}
	if since > 0 {
		return fmt.Sprintf("%s for %s (%d %s, last %s):", kind, scope, count, label, since)
	}
	return fmt.Sprintf("%s for %s (%d %s):", kind, scope, count, label)
}

func renderMilestone(index int, item milestone) []string {
	title := fmt.Sprintf("%s [%s] %s", styleIndex(fmt.Sprintf("%d.", index)), shortHistoryTS(item.Timestamp), styleTarget(item.Project+"/"+item.Task))
	body := firstNonEmpty(item.Title, item.Kind)
	lines := []string{title, "   " + body}
	if item.State != "" {
		lines = append(lines, "   "+styleLabel("State: ")+item.State)
	}
	if item.Next != "" {
		lines = append(lines, "   "+styleLabel("Next: ")+item.Next)
	}
	lines = append(lines, "   "+styleLabel("Source: ")+item.Source)
	return lines
}

func filterTimelineEvents(items []events.Event, projectFilter, taskFilter string, since time.Duration) []events.Event {
	var filtered []events.Event
	for _, item := range items {
		if projectFilter != "" && item.Project != "" && item.Project != projectFilter {
			continue
		}
		if taskFilter != "" && item.Task != taskFilter {
			continue
		}
		if since > 0 {
			ts, err := time.Parse(time.RFC3339, item.Timestamp)
			if err != nil || time.Since(ts) > since {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func renderTimelineEvent(item events.Event) string {
	target := timelineTarget(item)
	actor := firstNonEmpty(strings.TrimSpace(item.Agent), "unknown")
	host := firstNonEmpty(strings.TrimSpace(item.Host), "unknown-host")
	return fmt.Sprintf("[%s] %s %s by %s@%s", item.Timestamp, styleTarget(target), describeEvent(item), actor, host)
}

func timelineTarget(item events.Event) string {
	switch {
	case item.Project != "" && item.Task != "":
		return item.Project + "/" + item.Task
	case item.Project != "":
		return item.Project
	default:
		return "-"
	}
}

func describeEvent(item events.Event) string {
	if detail := strings.TrimSpace(item.Detail); detail != "" {
		switch item.Type {
		case "task_started":
			return "started task: " + detail
		case "capture", "capture_saved":
			return "captured progress: " + detail
		case "proposal":
			return "captured proposal: " + detail
		case "request":
			return "captured request: " + detail
		case "response":
			return "captured response: " + detail
		case "decision":
			return "captured decision: " + detail
		case "handoff":
			return "saved handoff: " + detail
		case "refresh":
			return "refreshed snapshot: " + detail
		case "clean":
			return "cleaned snapshots: " + detail
		case "task_closed":
			return "closed task: " + detail
		case "task_reopened":
			return "reopened task: " + detail
		case "task_deleted":
			return "deleted task: " + detail
		case "project_initialized":
			return "initialized project: " + detail
		case "project_deleted":
			return "deleted project: " + detail
		case "session_refreshed":
			return "refreshed continuum session: " + detail
		case "sync":
			return "synced continuum storage: " + detail
		case "repair":
			return "repaired storage: " + detail
		case "export":
			return "exported archive: " + detail
		case "import":
			return "imported archive: " + detail
		case "agent_install":
			return "installed bootstrap: " + detail
		case "agent_remove":
			return "removed bootstrap: " + detail
		}
		return item.Type + ": " + detail
	}
	return item.Type
}

func shortHistoryTS(value string) string {
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return ts.Format("2006-01-02 15:04")
}

func extractState(content string) string {
	items := parse.ExtractBulletList(content, "state")
	var out []string
	for i, item := range items {
		if i >= 2 {
			break
		}
		item = strings.TrimSpace(item)
		if item != "" && item != "..." {
			out = append(out, item)
		}
	}
	return strings.Join(out, " | ")
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "..." {
			return value
		}
	}
	return ""
}

func styleHeader(value string) string {
	if !historyColorsEnabled() {
		return value
	}
	return historyHeaderStyle + value + historyResetStyle
}

func styleIndex(value string) string {
	if !historyColorsEnabled() {
		return value
	}
	return historyIndexStyle + value + historyResetStyle
}

func styleTarget(value string) string {
	if !historyColorsEnabled() {
		return value
	}
	return historyTargetStyle + value + historyResetStyle
}

func styleLabel(value string) string {
	if !historyColorsEnabled() {
		return value
	}
	return historyLabelStyle + value + historyResetStyle
}

func historyColorsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if term := os.Getenv("TERM"); term == "" || term == "dumb" {
		return false
	}
	return prompt.IsInteractiveOutput()
}

func ansiStyle(codes ...string) string {
	return prompt.AnsiStyle(codes...)
}

func visibleWidth(value string) int {
	return prompt.VisibleWidth(value)
}
