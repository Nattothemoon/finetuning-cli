// Package config handles reading and writing the JSON config file used by the
// CLI. Paths are derived from os.UserConfigDir so that we land in:
//
//   - macOS:   ~/Library/Application Support/finetuning/config.json
//   - Linux:   ~/.config/finetuning/config.json
//   - Windows: %APPDATA%\finetuning\config.json
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const AppDir = "finetuning"

// Config is the on-disk JSON config.
type Config struct {
	BaseURL         string `json:"baseUrl,omitempty"`
	DefaultDuration int    `json:"defaultDuration,omitempty"`
	DefaultBPM      int    `json:"defaultBpm,omitempty"`
}

// Default returns the baseline config (used when no file exists).
func Default() Config {
	return Config{
		BaseURL:         "https://pub.finetuning.ai",
		DefaultDuration: 60,
		DefaultBPM:      120,
	}
}

// Dir returns the directory the CLI writes config + credential fallback into.
// Override the location by setting FINETUNING_CONFIG_HOME.
func Dir() (string, error) {
	if override := os.Getenv("FINETUNING_CONFIG_HOME"); override != "" {
		return override, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(base, AppDir), nil
}

// Path returns the full config.json path.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from disk. If the file is absent, Default() is returned
// and no error — a fresh install is a normal state.
func Load() (Config, error) {
	cfg := Default()
	p, err := Path()
	if err != nil {
		return cfg, err
	}
	buf, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(buf, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = Default().BaseURL
	}
	return cfg, nil
}

// Save writes the config to disk, creating parent dirs as needed.
func Save(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	buf, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(dir, "config.json")
	return os.WriteFile(p, buf, 0o600)
}
