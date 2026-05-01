package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"continuum/internal/filestore"
	"continuum/internal/setup"
)

type Artifact struct {
	Name string
	Type filestore.CaptureType
}

var collaborationTypes = []filestore.CaptureType{
	filestore.ProposalCapture,
	filestore.RequestCapture,
	filestore.ResponseCapture,
	filestore.DecisionCapture,
}

func ListArtifacts(task, project string, captureType string) ([]Artifact, error) {
	taskDir, err := resolveTaskDir(task, project)
	if err != nil {
		return nil, err
	}
	types, err := resolveArtifactTypes(captureType)
	if err != nil {
		return nil, err
	}

	var artifacts []Artifact
	for _, typ := range types {
		paths, err := filestore.AllCapturesOfType(taskDir, typ)
		if err != nil {
			return nil, fmt.Errorf("cannot list %s artifacts: %w", typ, err)
		}
		for _, path := range paths {
			artifacts = append(artifacts, Artifact{
				Name: filepath.Base(path),
				Type: typ,
			})
		}
	}
	sort.SliceStable(artifacts, func(i, j int) bool {
		return artifacts[i].Name < artifacts[j].Name
	})
	return artifacts, nil
}

func ReadArtifact(task, project, name string) (string, error) {
	taskDir, err := resolveTaskDir(task, project)
	if err != nil {
		return "", err
	}
	if err := validateArtifactName(name); err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(taskDir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("artifact %q not found", name)
		}
		return "", fmt.Errorf("cannot read artifact %q: %w", name, err)
	}
	return string(data), nil
}

func ResolveArtifact(task, project, name string) error {
	taskDir, err := resolveTaskDir(task, project)
	if err != nil {
		return err
	}
	if err := validateArtifactName(name); err != nil {
		return err
	}
	if err := filestore.ResolveArtifact(taskDir, name); err != nil {
		return err
	}
	files := []string{
		taskFile(project, task, name),
		taskFile(project, task, filepath.ToSlash(filepath.Join("resolved", name))),
	}
	if err := commitTaskWrite(project, task, "resolve", "artifact resolved", files); err != nil {
		return err
	}
	return nil
}

func resolveArtifactWithoutCommit(task, project, name string) error {
	taskDir, err := resolveTaskDir(task, project)
	if err != nil {
		return err
	}
	if err := validateArtifactName(name); err != nil {
		return err
	}
	return filestore.ResolveArtifact(taskDir, name)
}

func artifactFile(project, task, name string) string {
	return taskFile(project, task, name)
}

func resolvedArtifactFile(project, task, name string) string {
	return taskFile(project, task, filepath.ToSlash(filepath.Join("resolved", name)))
}

func resolveTaskDir(task, project string) (string, error) {
	if err := setup.ValidateTaskName(task); err != nil {
		return "", err
	}
	if err := setup.ValidateProjectName(project); err != nil {
		return "", err
	}
	taskDir := filepath.Join(setup.ContinuumPath(), "projects", project, "tasks", task)
	if info, err := os.Stat(taskDir); err != nil || !info.IsDir() {
		if os.IsNotExist(err) || err == nil {
			return "", fmt.Errorf("task %q not found in project %q", task, project)
		}
		return "", fmt.Errorf("cannot access task %q in project %q: %w", task, project, err)
	}
	return taskDir, nil
}

func resolveArtifactTypes(value string) ([]filestore.CaptureType, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "all" {
		return collaborationTypes, nil
	}
	captureType, err := filestore.ValidateCaptureType(value)
	if err != nil {
		return nil, err
	}
	if captureType == filestore.StateCapture {
		return nil, fmt.Errorf("artifact type must be proposal, request, response, decision, or all")
	}
	return []filestore.CaptureType{captureType}, nil
}

func validateArtifactName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("artifact filename is required")
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("artifact filename must not contain path separators")
	}
	for _, typ := range collaborationTypes {
		prefix := string(typ) + "."
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".md") {
			return nil
		}
	}
	return fmt.Errorf("artifact filename must be proposal, request, response, or decision markdown")
}

func validateResolvableArtifactName(name string) error {
	if err := validateArtifactName(name); err != nil {
		return err
	}
	for _, prefix := range []string{"proposal.", "request."} {
		if strings.HasPrefix(name, prefix) {
			return nil
		}
	}
	return fmt.Errorf("resolved artifact must be a proposal or request")
}
