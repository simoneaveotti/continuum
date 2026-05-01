package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostName_PrefersEnvironment(t *testing.T) {
	t.Setenv("CONTINUUM_HOST", "ci-host")
	if got := HostName(); got != "ci-host" {
		t.Fatalf("HostName() = %q, want %q", got, "ci-host")
	}
}

func TestHostName_UsesConfiguredHost(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)
	t.Setenv("CONTINUUM_HOST", "")
	if err := SetHost("workstation-1"); err != nil {
		t.Fatalf("SetHost() error: %v", err)
	}
	if got := HostName(); got != "workstation-1" {
		t.Fatalf("HostName() = %q, want %q", got, "workstation-1")
	}
}

func TestSetHost_WritesLocalIdentityConfig(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CONTINUUM_PATH", base)
	if err := SetHost("office-pc"); err != nil {
		t.Fatalf("SetHost() error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "local", "identity.json")); err != nil {
		t.Fatalf("expected identity.json to exist: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Host != "office-pc" {
		t.Fatalf("Load().Host = %q, want %q", cfg.Host, "office-pc")
	}
}

func TestAgentName_UsesContinuumAgent(t *testing.T) {
	t.Setenv("CONTINUUM_AGENT", "codex")
	if got := AgentName(); got != "codex" {
		t.Fatalf("AgentName() = %q, want %q", got, "codex")
	}
}
