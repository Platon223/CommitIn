// Package config reads and writes the local cmtin config file, which holds
// the session token and (later) other per-machine settings.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the on-disk shape of ~/.config/cmtin/config.json (or
// $XDG_CONFIG_HOME/cmtin/config.json).
type Config struct {
	Token  string `json:"token,omitempty"`
	APIURL string `json:"api_url,omitempty"`
}

// Path returns the config file's location, honoring XDG_CONFIG_HOME.
func Path() (string, error) {
	dir, err := dirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func dirPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "cmtin"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "cmtin"), nil
}

// Load reads the config file. A missing file is not an error: it returns a
// zero-value Config, since that's the normal state before the first login.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config at %s: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse config at %s: %w", path, err)
	}
	return &c, nil
}

// Save writes the config file, creating its directory if needed. The file is
// created with 0600 since it holds a bearer token.
func Save(c *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write config at %s: %w", path, err)
	}
	return nil
}
