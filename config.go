package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RMSSDir      string `yaml:"rmss_dir"`
	StateFile    string `yaml:"state_file"`
	PollInterval int    `yaml:"poll_interval"`
}

func DefaultConfig() *Config {
	return &Config{
		RMSSDir:      "~/.config/rmss",
		StateFile:    "~/.config/rmss/state.yaml",
		PollInterval: 3,
	}
}

func loadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultConfig(), nil
	}

	configPath := filepath.Join(home, ".config", "rmss", "configs.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	config.RMSSDir = expandPath(config.RMSSDir)
	config.StateFile = expandPath(config.StateFile)

	return config, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return os.ExpandEnv(path)
}
