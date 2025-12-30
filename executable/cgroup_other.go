//go:build !linux

package executable

// cgroupManager is a no-op on non-Linux platforms
type cgroupManager struct{}

// newCgroupManager returns a no-op manager on non-Linux platforms
func newCgroupManager(memoryLimitBytes int64, pid int) (*cgroupManager, error) {
	return &cgroupManager{}, nil
}

// wasOOMKilled always returns false on non-Linux platforms
func (c *cgroupManager) wasOOMKilled() bool {
	return false
}

// cleanup is a no-op on non-Linux platforms
func (c *cgroupManager) cleanup() {}

