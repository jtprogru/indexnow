// Package config loads the optional yaml config file that provides
// per-host defaults for the indexnow CLI. The file is searched at
// $XDG_CONFIG_HOME/indexnow/config.yaml (falling back to
// $HOME/.config/indexnow/config.yaml) unless an explicit path is given.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Host        string `yaml:"host"`
	Key         string `yaml:"key"`
	KeyLocation string `yaml:"key_location"`
	Endpoint    string `yaml:"endpoint"`
	UserAgent   string `yaml:"user_agent"`
}

// Load reads and parses the yaml config at path. An empty file is
// returned as a zero-valued Config. Unknown fields are rejected so
// typos surface early.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// DefaultPath returns the conventional location for the indexnow config
// file. Empty string is returned only when the home directory cannot be
// resolved, in which case callers should treat the default as absent.
func DefaultPath() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "indexnow", "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "indexnow", "config.yaml")
}
