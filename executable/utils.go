package executable

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/mattn/go-isatty"
)

func GetMemoryLimitInBytes() int64 {
	// 2 GB by default
	memoryLimitInBytes := int64(2*1024) * 1024 * 1024
	memoryLimitEnvVar := os.Getenv("EXECUTABLE_MEMORY_LIMIT_IN_MB")

	if memoryLimitEnvVar == "" {
		return memoryLimitInBytes
	}

	convertedMemoryLimitInMb, err := strconv.Atoi(memoryLimitEnvVar)

	// Panic if the variable is set but is not a number - should be notified
	if err != nil {
		panic("Codecrafters Internal Error - EXECUTABLE_MEMORY_LIMIT_IN_MB is not an integer")
	}

	if convertedMemoryLimitInMb < 0 {
		panic(fmt.Sprintf("Codecrafters Internal Error - EXECUTABLE_MEMORY_LIMIT_IN_MB is negative: %d", convertedMemoryLimitInMb))
	}

	return int64(convertedMemoryLimitInMb) * 1024 * 1024
}

// isTTY returns true if the object is a tty
func isTTY(o any) bool {
	file, ok := o.(*os.File)
	if !ok {
		return false
	}

	return isatty.IsTerminal(file.Fd())
}

// ResolveAbsolutePath resolves the path according the following rules:
// 1. If executable is not found ('path' is neither in $PATH, nor found at the path specified) -> Error is returned
// 2. If the 'path' contains slash, its absolute path is returned
// 3. If the 'path' does not contains a slash, it is searched for in PATH and its absolute path
func resolveAbsolutePath(path string) (absolutePath string, err error) {
	executablePath, err := exec.LookPath(path)

	// exec.LeookPath() failed: Try filepath.Abs()
	if err != nil {
		return filepath.Abs(path)
	}

	// No error: The executable was found
	// 1. If 'path' was a comand, 'executablePath' is the absolute path of that command
	// 2. If 'path' was a relative path to the executable, 'executablePath' is the relative path to that executable
	// So, we convert the relative path to the absolute path
	return filepath.Abs(executablePath)
}
