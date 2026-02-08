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
	if dir := os.Getenv("LAZYRMSS_CONFIG_DIR"); dir != "" {
		return dir
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "lazyrmss")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "lazyrmss")
}

func dataDir() string {
	if dir := os.Getenv("LAZYRMSS_DATA_DIR"); dir != "" {
		return dir
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "lazyrmss")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "lazyrmss")
}

func (a *App) stateFilePath() string {
	return filepath.Join(dataDir(), "state.yaml")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	home, _ := os.UserHomeDir()
	if strings.Contains(path, "$XDG_CONFIG_HOME") && os.Getenv("XDG_CONFIG_HOME") == "" {
		path = strings.ReplaceAll(path, "$XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		return path
	}
	if strings.Contains(path, "$XDG_DATA_HOME") && os.Getenv("XDG_DATA_HOME") == "" {
		path = strings.ReplaceAll(path, "$XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
		return path
	}
	return os.ExpandEnv(path)
}
