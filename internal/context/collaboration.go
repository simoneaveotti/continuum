package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"continuum/internal/filestore"
	"continuum/internal/parse"
	"continuum/internal/setup"
)

// CollaborationArtifacts summarizes the open typed captures for a task.
type CollaborationArtifacts struct {
	ProposalCount  int
	RequestCount   int
	LatestProposal string
	LatestRequest  string
	LatestResponse string
	LatestDecision string
}

func appendCompactCollaboration(lines *[]string, taskName, project string) {
	artifacts, err := LoadCollaborationArtifacts(taskName, project)
	if err != nil {
		return
	}
	var parts []string
	if artifacts.ProposalCount > 0 {
		parts = append(parts, fmt.Sprintf("proposals=%d", artifacts.ProposalCount))
	}
	if artifacts.RequestCount > 0 {
		parts = append(parts, fmt.Sprintf("requests=%d", artifacts.RequestCount))
	}
	if len(parts) > 0 {
		*lines = append(*lines, "OPEN:"+strings.Join(parts, " | "))
	}
	if value := cleanCompactValue(artifacts.LatestResponse); value != "" {
		*lines = append(*lines, "RESP:"+value)
	}
	if value := cleanCompactValue(artifacts.LatestDecision); value != "" {
		*lines = append(*lines, "DECISION:"+value)
	}
}

// LoadCollaborationArtifacts reads the latest open typed captures for a task.
func LoadCollaborationArtifacts(taskName, project string) (*CollaborationArtifacts, error) {
	if err := setup.ValidateTaskName(taskName); err != nil {
		return nil, err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return nil, err
	}
	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", taskName)
	artifacts := &CollaborationArtifacts{}

	proposals, err := filestore.AllCapturesOfType(taskDir, filestore.ProposalCapture)
	if err != nil {
		return nil, err
	}
	requests, err := filestore.AllCapturesOfType(taskDir, filestore.RequestCapture)
	if err != nil {
		return nil, err
	}
	artifacts.ProposalCount = len(proposals)
	artifacts.RequestCount = len(requests)
	if len(proposals) > 0 {
		artifacts.LatestProposal = latestArtifactSummary(proposals[len(proposals)-1])
	}
	if len(requests) > 0 {
		artifacts.LatestRequest = latestArtifactSummary(requests[len(requests)-1])
	}
	if path, _, err := filestore.LatestCaptureOfType(taskDir, filestore.ResponseCapture); err == nil && path != "" {
		artifacts.LatestResponse = latestArtifactSummary(path)
	} else if err != nil {
		return nil, err
	}
	if path, _, err := filestore.LatestCaptureOfType(taskDir, filestore.DecisionCapture); err == nil && path != "" {
		artifacts.LatestDecision = latestArtifactSummary(path)
	} else if err != nil {
		return nil, err
	}
	return artifacts, nil
}

func appendCollaboration(lines []string, taskName, project string) []string {
	artifacts, err := LoadCollaborationArtifacts(taskName, project)
	if err != nil {
		return lines
	}
	if artifacts.ProposalCount > 0 {
		lines = append(lines, fmt.Sprintf("OPEN PROPOSALS: %d (latest: %s)", artifacts.ProposalCount, artifacts.LatestProposal))
	}
	if artifacts.RequestCount > 0 {
		lines = append(lines, fmt.Sprintf("OPEN REQUESTS: %d (latest: %s)", artifacts.RequestCount, artifacts.LatestRequest))
	}
	if artifacts.LatestResponse != "" {
		lines = append(lines, fmt.Sprintf("LATEST RESPONSE: %s", artifacts.LatestResponse))
	}
	if artifacts.LatestDecision != "" {
		lines = append(lines, fmt.Sprintf("LATEST DECISION: %s", artifacts.LatestDecision))
	}
	return lines
}

func latestArtifactSummary(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "unreadable artifact"
	}
	body := artifactBody(string(data))
	for _, section := range []string{"decision", "recommendation", "response", "request", "proposal"} {
		if value := parse.ExtractField(body, section); value != "" && value != "..." {
			return compactSummary(value)
		}
	}
	return compactSummary(firstUserArtifactLine(body))
}

func firstUserArtifactLine(content string) string {
	for _, line := range splitLines(content) {
		line = trimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.EqualFold(line, "## Last Updated") {
			break
		}
		return line
	}
	return "content provided"
}

func artifactBody(content string) string {
	lines := splitLines(content)
	for i, line := range lines {
		if strings.EqualFold(trimSpace(line), "## Capture Type") {
			start := i + 1
			for start < len(lines) && trimSpace(lines[start]) != "" {
				start++
			}
			for start < len(lines) && trimSpace(lines[start]) == "" {
				start++
			}
			return strings.Join(lines[start:], "\n")
		}
	}
	return content
}
