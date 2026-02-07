package main

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type OptionState struct {
	Enabled bool     `yaml:"enabled"`
	Addons  []string `yaml:"addons"`
}

func (a *App) loadState() error {
	data, err := os.ReadFile(a.stateFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var state map[string]map[string]OptionState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return err
	}

	for catName, catState := range state {
		options, ok := a.options[catName]
		if !ok {
			continue
		}
		for _, opt := range options {
			if optState, ok := catState[opt.Name]; ok {
				opt.Enabled = optState.Enabled
				for _, addonName := range optState.Addons {
					opt.ActiveAddons[addonName] = true
				}
			}
		}
	}

	return nil
}

func (a *App) saveState() error {
	state := make(map[string]map[string]OptionState)

	for _, cat := range a.categories {
		catState := make(map[string]OptionState)
		for _, opt := range a.options[cat.Name] {
			var addons []string
			for _, addon := range opt.Addons {
				if opt.ActiveAddons[addon.Name] {
					addons = append(addons, addon.Name)
				}
			}
			sort.Strings(addons)
			catState[opt.Name] = OptionState{
				Enabled: opt.Enabled,
				Addons:  addons,
			}
		}
		state[cat.Name] = catState
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}

	os.MkdirAll(filepath.Dir(a.stateFilePath()), 0755)
	return os.WriteFile(a.stateFilePath(), data, 0644)
}
