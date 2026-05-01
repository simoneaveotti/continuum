package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Host string `json:"host,omitempty"`
}

func AgentName() string {
	if value := strings.TrimSpace(os.Getenv("CONTINUUM_AGENT")); value != "" {
		return value
	}
	return "unknown"
}

func HostName() string {
	if value := strings.TrimSpace(os.Getenv("CONTINUUM_HOST")); value != "" {
		return value
	}
	if cfg, err := Load(); err == nil {
		if value := strings.TrimSpace(cfg.Host); value != "" {
			return value
		}
	}
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return "unknown-host"
	}
	return host
}

func Load() (Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read identity config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse identity config: %w", err)
	}
	return cfg, nil
}

func SetHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.Host = host
	if err := os.MkdirAll(filepath.Dir(configPath()), 0o755); err != nil {
		return fmt.Errorf("create local config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode identity config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(configPath(), data, 0o644); err != nil {
		return fmt.Errorf("write identity config: %w", err)
	}
	return nil
}

func configPath() string {
	return filepath.Join(continuumPath(), "local", "identity.json")
}

func continuumPath() string {
	if path := os.Getenv("CONTINUUM_PATH"); path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, ".continuum")
	}
	return filepath.Join(home, ".continuum")
}
