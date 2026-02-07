package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// --- Discovery ---

func (a *App) discoverAll() error {
	entries, err := os.ReadDir(a.config.RMSSDir)
	if err != nil {
		return fmt.Errorf("reading rmss dir %s: %w", a.config.RMSSDir, err)
	}

	a.categories = nil
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		cat := Category{
			Name: entry.Name(),
			Dir:  filepath.Join(a.config.RMSSDir, entry.Name()),
		}
		a.categories = append(a.categories, cat)

		opts, err := discoverOptions(cat)
		if err != nil {
			continue
		}
		a.options[cat.Name] = opts
	}

	sort.Slice(a.categories, func(i, j int) bool {
		return a.categories[i].Name < a.categories[j].Name
	})

	return nil
}

func discoverOptions(cat Category) ([]*Option, error) {
	entries, err := os.ReadDir(cat.Dir)
	if err != nil {
		return nil, err
	}

	var options []*Option
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		opt := &Option{
			Name:         entry.Name(),
			Dir:          filepath.Join(cat.Dir, entry.Name()),
			Category:     cat.Name,
			ActiveAddons: make(map[string]bool),
		}

		files, err := os.ReadDir(opt.Dir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
				continue
			}
			fullPath := filepath.Join(opt.Dir, f.Name())
			name := strings.TrimSuffix(f.Name(), ".yaml")
			if name == "base" {
				opt.BaseFile = fullPath
			} else {
				label, color := getAddonDisplay(name)
				opt.Addons = append(opt.Addons, Addon{
					Name:  name,
					File:  fullPath,
					Label: label,
					Color: color,
				})
			}
		}

		sort.Slice(opt.Addons, func(i, j int) bool {
			return opt.Addons[i].Name < opt.Addons[j].Name
		})

		if opt.BaseFile != "" {
			options = append(options, opt)
		}
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Name < options[j].Name
	})

	return options, nil
}

// --- YAML loading and merging ---

func loadYAMLFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// deepMerge merges src into dst recursively.
// Maps merge recursively, lists append, scalars from src override dst.
func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = make(map[string]interface{})
	}
	for key, srcVal := range src {
		dstVal, exists := dst[key]
		if !exists {
			dst[key] = srcVal
			continue
		}

		srcMap, srcIsMap := srcVal.(map[string]interface{})
		dstMap, dstIsMap := dstVal.(map[string]interface{})
		if srcIsMap && dstIsMap {
			dst[key] = deepMerge(dstMap, srcMap)
			continue
		}

		srcList, srcIsList := srcVal.([]interface{})
		dstList, dstIsList := dstVal.([]interface{})
		if srcIsList && dstIsList {
			dst[key] = append(dstList, srcList...)
			continue
		}

		dst[key] = srcVal
	}
	return dst
}

func resolveOption(opt *Option) (map[string]interface{}, error) {
	result, err := loadYAMLFile(opt.BaseFile)
	if err != nil {
		return nil, fmt.Errorf("loading base for %s: %w", opt.Name, err)
	}

	for _, addon := range opt.Addons {
		if !opt.ActiveAddons[addon.Name] {
			continue
		}
		addonData, err := loadYAMLFile(addon.File)
		if err != nil {
			continue
		}
		result = deepMerge(result, addonData)
	}

	return result, nil
}

func (a *App) buildGlobalCompose() (map[string]interface{}, error) {
	global := make(map[string]interface{})

	for _, cat := range a.categories {
		opts := a.options[cat.Name]
		for _, opt := range opts {
			if !opt.Enabled {
				continue
			}
			resolved, err := resolveOption(opt)
			if err != nil {
				continue
			}
			global = deepMerge(global, resolved)
		}
	}

	return global, nil
}

func renderYAML(data map[string]interface{}) (string, error) {
	out, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// --- Docker Compose execution ---

func (a *App) dockerComposeUp() {
	global, err := a.buildGlobalCompose()
	if err != nil {
		return
	}

	yamlBytes, err := yaml.Marshal(global)
	if err != nil {
		return
	}

	tmpFile, err := os.CreateTemp("", "lazyrmss-compose-*.yaml")
	if err != nil {
		return
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(yamlBytes); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return
	}
	tmpFile.Close()

	a.app.Suspend(func() {
		cmd := exec.Command("docker", "compose", "-f", tmpPath, "up", "-d")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		fmt.Println("\nPress Enter to return to LazyRMSS...")
		fmt.Scanln()
		os.Remove(tmpPath)
	})
}

func (a *App) dockerComposeDown() {
	global, err := a.buildGlobalCompose()
	if err != nil {
		return
	}

	yamlBytes, err := yaml.Marshal(global)
	if err != nil {
		return
	}

	tmpFile, err := os.CreateTemp("", "lazyrmss-compose-*.yaml")
	if err != nil {
		return
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(yamlBytes); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return
	}
	tmpFile.Close()

	a.app.Suspend(func() {
		cmd := exec.Command("docker", "compose", "-f", tmpPath, "down")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		fmt.Println("\nPress Enter to return to LazyRMSS...")
		fmt.Scanln()
		os.Remove(tmpPath)
	})
}
