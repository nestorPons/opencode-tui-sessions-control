package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	SourceDB     string
	AppDB        string
	ExportsDir   string
	EffectiveGap time.Duration
	DayGap       time.Duration
	Internal     []string
}

func Default() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("home directory: %w", err)
	}
	cfg := Config{
		SourceDB:     filepath.Join(home, ".local", "share", "opencode", "opencode.db"),
		AppDB:        filepath.Join(home, ".local", "share", "sessions-control", "sessions.db"),
		ExportsDir:   filepath.Join(home, "sessions-control-exports"),
		EffectiveGap: 15 * time.Minute,
		DayGap:       5 * time.Minute,
		Internal:     []string{"global", "sessions-control"},
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
	if v := os.Getenv("SESSIONS_CONTROL_GAP"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.EffectiveGap = time.Duration(n) * time.Minute
		}
	}
	if v := os.Getenv("SESSIONS_CONTROL_DAY_GAP"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.DayGap = time.Duration(n) * time.Minute
		}
	}
	if v := os.Getenv("SESSIONS_CONTROL_INTERNAL"); v != "" {
		cfg.Internal = splitList(v)
	}
	return cfg, nil
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
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
