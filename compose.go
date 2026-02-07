package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rivo/tview"
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

// --- Resource name extraction ---

func extractContainerNames(resolved map[string]interface{}) []string {
	services, ok := resolved["services"].(map[string]interface{})
	if !ok {
		return nil
	}
	var names []string
	for svcKey, svcVal := range services {
		svcMap, ok := svcVal.(map[string]interface{})
		if !ok {
			names = append(names, svcKey)
			continue
		}
		if cn, ok := svcMap["container_name"].(string); ok && cn != "" {
			names = append(names, cn)
		} else {
			names = append(names, svcKey)
		}
	}
	return names
}

func extractNetworkNames(resolved map[string]interface{}) []string {
	networks, ok := resolved["networks"].(map[string]interface{})
	if !ok {
		return nil
	}
	var names []string
	for netKey, netVal := range networks {
		netMap, ok := netVal.(map[string]interface{})
		if !ok {
			names = append(names, netKey)
			continue
		}
		if n, ok := netMap["name"].(string); ok && n != "" {
			names = append(names, n)
		} else {
			names = append(names, netKey)
		}
	}
	return names
}

func extractVolumeNames(resolved map[string]interface{}) []string {
	volumes, ok := resolved["volumes"].(map[string]interface{})
	if !ok {
		return nil
	}
	var names []string
	for volKey, volVal := range volumes {
		volMap, ok := volVal.(map[string]interface{})
		if !ok {
			names = append(names, volKey)
			continue
		}
		if n, ok := volMap["name"].(string); ok && n != "" {
			names = append(names, n)
		} else {
			names = append(names, volKey)
		}
	}
	return names
}

func (a *App) isOptionRunning(opt *Option) bool {
	if a.dockerStatus == nil {
		return false
	}

	resolved, err := resolveOption(opt)
	if err != nil {
		return false
	}

	// Check based on the option's category
	for _, name := range extractContainerNames(resolved) {
		if a.dockerStatus.IsContainerRunning(name) {
			return true
		}
	}
	for _, name := range extractNetworkNames(resolved) {
		if a.dockerStatus.IsNetworkExists(name) {
			return true
		}
	}
	for _, name := range extractVolumeNames(resolved) {
		if a.dockerStatus.IsVolumeExists(name) {
			return true
		}
	}

	return false
}

// --- Docker Compose execution ---

type logWriter struct {
	app  *tview.Application
	view *tview.TextView
	ansi io.Writer
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.app.QueueUpdateDraw(func() {
		w.ansi.Write(p)
		w.view.ScrollToEnd()
	})
	return len(p), nil
}

func (a *App) runDockerCompose(composeData map[string]interface{}, args ...string) {
	yamlBytes, err := yaml.Marshal(composeData)
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

	cmdStr := strings.Join(args, " ")
	a.logView.Clear()
	fmt.Fprintf(a.logView, "[yellow]$ docker compose %s[-]\n", cmdStr)

	cmdArgs := append([]string{"compose", "-f", tmpPath}, args...)
	cmd := exec.Command("docker", cmdArgs...)

	writer := &logWriter{
		app:  a.app,
		view: a.logView,
		ansi: tview.ANSIWriter(a.logView),
	}
	cmd.Stdout = writer
	cmd.Stderr = writer

	go func() {
		err := cmd.Run()
		os.Remove(tmpPath)
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				fmt.Fprintf(a.logView, "\n[red]✗ %v[-]\n", err)
			} else {
				fmt.Fprintf(a.logView, "\n[green]✓ Done[-]\n")
			}
			a.logView.ScrollToEnd()
		})
		a.refreshDockerStatus()
	}()
}

func (a *App) dockerComposeGlobal(args ...string) {
	global, err := a.buildGlobalCompose()
	if err != nil {
		return
	}
	a.runDockerCompose(global, args...)
}

func (a *App) runDockerDirect(targets []string, args ...string) {
	cmdArgs := append(args, targets...)
	cmdStr := strings.Join(cmdArgs, " ")
	a.logView.Clear()
	fmt.Fprintf(a.logView, "[yellow]$ docker %s[-]\n", cmdStr)

	cmd := exec.Command("docker", cmdArgs...)

	writer := &logWriter{
		app:  a.app,
		view: a.logView,
		ansi: tview.ANSIWriter(a.logView),
	}
	cmd.Stdout = writer
	cmd.Stderr = writer

	go func() {
		err := cmd.Run()
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				fmt.Fprintf(a.logView, "\n[red]✗ %v[-]\n", err)
			} else {
				fmt.Fprintf(a.logView, "\n[green]✓ Done[-]\n")
			}
			a.logView.ScrollToEnd()
		})
		a.refreshDockerStatus()
	}()
}

func (a *App) dockerDirectSingle(args ...string) {
	opt := a.getSelectedOption()
	if opt == nil {
		return
	}
	resolved, err := resolveOption(opt)
	if err != nil {
		return
	}
	names := extractContainerNames(resolved)
	if len(names) == 0 {
		return
	}
	a.runDockerDirect(names, args...)
}

func extractImageNames(resolved map[string]interface{}) []string {
	services, ok := resolved["services"].(map[string]interface{})
	if !ok {
		return nil
	}
	var names []string
	for _, svcVal := range services {
		svcMap, ok := svcVal.(map[string]interface{})
		if !ok {
			continue
		}
		if img, ok := svcMap["image"].(string); ok && img != "" {
			names = append(names, img)
		}
	}
	return names
}

func (a *App) dockerPullSingle() {
	opt := a.getSelectedOption()
	if opt == nil {
		return
	}
	resolved, err := resolveOption(opt)
	if err != nil {
		return
	}
	images := extractImageNames(resolved)
	if len(images) == 0 {
		return
	}
	a.runDockerDirect(images, "pull")
}

func (a *App) refreshDockerStatus() {
	if a.dockerStatus == nil {
		return
	}
	go func() {
		a.dockerStatus.poll(context.Background())
		a.app.QueueUpdateDraw(func() {
			a.refreshOptionsList()
		})
	}()
}
