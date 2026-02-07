package main

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// DockerStatus holds the live state of Docker resources, protected by a mutex.
type DockerStatus struct {
	mu sync.RWMutex

	RunningContainers map[string]bool
	ExistingNetworks  map[string]bool
	ExistingVolumes   map[string]bool
}

func queryRunningContainers(ctx context.Context) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseLines(string(out)), nil
}

func queryExistingNetworks(ctx context.Context) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "network", "ls", "--format", "{{.Name}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseLines(string(out)), nil
}

func queryExistingVolumes(ctx context.Context) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "volume", "ls", "--format", "{{.Name}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseLines(string(out)), nil
}

func parseLines(output string) map[string]bool {
	result := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result[line] = true
		}
	}
	return result
}

func (ds *DockerStatus) poll(ctx context.Context) {
	containers, errC := queryRunningContainers(ctx)
	networks, errN := queryExistingNetworks(ctx)
	volumes, errV := queryExistingVolumes(ctx)

	ds.mu.Lock()
	defer ds.mu.Unlock()

	if errC == nil {
		ds.RunningContainers = containers
	}
	if errN == nil {
		ds.ExistingNetworks = networks
	}
	if errV == nil {
		ds.ExistingVolumes = volumes
	}
}

// StartPolling launches a background goroutine that polls Docker status
// at the given interval and calls refreshUI after each poll.
func (ds *DockerStatus) StartPolling(ctx context.Context, interval time.Duration, refreshUI func()) {
	go func() {
		// Initial poll before entering the ticker loop.
		ds.poll(ctx)
		refreshUI()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ds.poll(ctx)
				refreshUI()
			}
		}
	}()
}

func (ds *DockerStatus) IsContainerRunning(name string) bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.RunningContainers[name]
}

func (ds *DockerStatus) IsNetworkExists(name string) bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.ExistingNetworks[name]
}

func (ds *DockerStatus) IsVolumeExists(name string) bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.ExistingVolumes[name]
}
