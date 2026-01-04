//go:build !linux

package executable

import "syscall"

// memoryMonitor is a no-op on non-Linux platforms
type memoryMonitor struct{}

// newMemoryMonitor returns a no-op monitor on non-Linux platforms
func newMemoryMonitor(memoryLimitBytes int64) *memoryMonitor {
	return &memoryMonitor{}
}

// start is a no-op on non-Linux platforms
func (m *memoryMonitor) start(pid int) {}

// wasOOMKilled always returns false on non-Linux platforms
func (m *memoryMonitor) wasOOMKilled() bool {
	return false
}

// stop is a no-op on non-Linux platforms
func (m *memoryMonitor) stop() {}

// applyHardMemoryLimit is a no-op on non-Linux platforms
func (m *memoryMonitor) applyHardMemoryLimit(sysProcAttr *syscall.SysProcAttr) {
	// Memory limiting via RLIMIT_AS is only supported on Linux
}
