package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	SourceDB   string
	AppDB      string
	ExportsDir string
}

func Default() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("home directory: %w", err)
	}
	cfg := Config{
		SourceDB:   filepath.Join(home, ".local", "share", "opencode", "opencode.db"),
		AppDB:      filepath.Join(home, ".local", "share", "sessions-control", "sessions.db"),
		ExportsDir: filepath.Join(home, "sessions-control-exports"),
	}
	if v := os.Getenv("SESSIONS_CONTROL_SOURCE"); v != "" {
		cfg.SourceDB = v
	}
	if v := os.Getenv("SESSIONS_CONTROL_DB"); v != "" {
		cfg.AppDB = v
	}
	if v := os.Getenv("SESSIONS_CONTROL_EXPORTS"); v != "" {
		cfg.ExportsDir = v
	}
	return cfg, nil
}

func (c Config) EnsureDirs() error {
	if err := os.MkdirAll(filepath.Dir(c.AppDB), 0o755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}
	if err := os.MkdirAll(c.ExportsDir, 0o755); err != nil {
		return fmt.Errorf("create exports dir: %w", err)
	}
	return nil
}
