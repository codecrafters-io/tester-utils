//go:build linux

package executable

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/containerd/cgroups/v3/cgroup2"
)

// cgroupManager handles cgroup-based resource limiting on Linux
type cgroupManager struct {
	manager       *cgroup2.Manager
	cgroupPath    string
	initialOOMKill uint64
}

// newCgroupManager creates a new cgroup with the specified memory limit
func newCgroupManager(memoryLimitBytes int64, pid int) (*cgroupManager, error) {
	if memoryLimitBytes <= 0 {
		return &cgroupManager{}, nil
	}

	// Create a unique cgroup path using PID and timestamp
	cgroupPath := fmt.Sprintf("/tester-utils-%d-%d", pid, time.Now().UnixNano())

	// Create cgroup2 resources with memory limit
	resources := &cgroup2.Resources{
		Memory: &cgroup2.Memory{
			Max: &memoryLimitBytes,
		},
	}

	// Create the cgroup manager
	manager, err := cgroup2.NewManager("/sys/fs/cgroup", cgroupPath, resources)
	if err != nil {
		return nil, fmt.Errorf("failed to create cgroup: %w", err)
	}

	// Add the process to the cgroup
	if err := manager.AddProc(uint64(pid)); err != nil {
		manager.Delete()
		return nil, fmt.Errorf("failed to add process to cgroup: %w", err)
	}

	// Read initial OOM kill count
	initialOOMKill := readOOMKillCount(cgroupPath)

	return &cgroupManager{
		manager:       manager,
		cgroupPath:    cgroupPath,
		initialOOMKill: initialOOMKill,
	}, nil
}

// wasOOMKilled checks if the process was killed due to exceeding memory limit
func (c *cgroupManager) wasOOMKilled() bool {
	if c.manager == nil {
		return false
	}

	currentOOMKill := readOOMKillCount(c.cgroupPath)
	return currentOOMKill > c.initialOOMKill
}

// cleanup removes the cgroup
func (c *cgroupManager) cleanup() {
	if c.manager != nil {
		c.manager.Delete()
		c.manager = nil
	}
}

// readOOMKillCount reads the oom_kill counter from memory.events
func readOOMKillCount(cgroupPath string) uint64 {
	eventsPath := filepath.Join("/sys/fs/cgroup", cgroupPath, "memory.events")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		return 0
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "oom_kill ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				count, _ := strconv.ParseUint(parts[1], 10, 64)
				return count
			}
		}
	}

	return 0
}

