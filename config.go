package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ResourcesDir string `yaml:"resources_dir"`
	PollInterval int    `yaml:"poll_interval"`
}

func DefaultConfig() *Config {
	return &Config{
		ResourcesDir: "$XDG_CONFIG_HOME/rmss",
		PollInterval: 3,
	}
}

func loadConfig() (*Config, error) {
	configPath := filepath.Join(configDir(), "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			config := DefaultConfig()
			config.ResourcesDir = expandPath(config.ResourcesDir)
			return config, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	config.ResourcesDir = expandPath(config.ResourcesDir)

	return config, nil
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "lazyrmss")
}

func (a *App) stateFilePath() string {
	return filepath.Join(configDir(), "state.yaml")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	if strings.Contains(path, "$XDG_CONFIG_HOME") && os.Getenv("XDG_CONFIG_HOME") == "" {
		home, _ := os.UserHomeDir()
		path = strings.ReplaceAll(path, "$XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		return path
	}
	return os.ExpandEnv(path)
}
