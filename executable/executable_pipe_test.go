package executable

import (
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	err := NewExecutable("/blah").Start()
	assertErrorContains(t, err, "not found")
	assertErrorContains(t, err, "blah")

	err = NewExecutable("./test_helpers/not_executable.sh").Start()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "not_executable.sh")

	err = NewExecutable("./test_helpers/haskell").Start()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "haskell")

	err = NewExecutable("./test_helpers/stdout_echo.sh").Start()
	assert.NoError(t, err)
}

func TestStartAndKill(t *testing.T) {
	e := NewExecutable("/blah")
	err := e.Start()
	assertErrorContains(t, err, "not found")
	assertErrorContains(t, err, "blah")
	err = e.Kill()
	assert.NoError(t, err)

	e = NewExecutable("./test_helpers/not_executable.sh")
	err = e.Start()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "not_executable.sh")
	err = e.Kill()
	assert.NoError(t, err)

	e = NewExecutable("./test_helpers/haskell")
	err = e.Start()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "haskell")
	err = e.Kill()
	assert.NoError(t, err)

	e = NewExecutable("./test_helpers/stdout_echo.sh")
	err = e.Start()
	assert.NoError(t, err)
	err = e.Kill()
	assert.NoError(t, err)
}

func assertErrorContains(t *testing.T, err error, expectedMsg string) {
	assert.Contains(t, err.Error(), expectedMsg)
}

func TestRun(t *testing.T) {
	e := NewExecutable("./test_helpers/stdout_echo.sh")
	result, err := e.Run("hey")
	assert.NoError(t, err)
	assert.Equal(t, "hey\n", string(result.Stdout))
}

func TestOutputCapture(t *testing.T) {
	// Stdout capture
	e := NewExecutable("./test_helpers/stdout_echo.sh")
	result, err := e.Run("hey")

	assert.NoError(t, err)
	assert.Equal(t, "hey\n", string(result.Stdout))
	assert.Equal(t, "", string(result.Stderr))

	// Stderr capture
	e = NewExecutable("./test_helpers/stderr_echo.sh")
	result, err = e.Run("hey")

	assert.NoError(t, err)
	assert.Equal(t, "", string(result.Stdout))
	assert.Equal(t, "hey\n", string(result.Stderr))
}

func TestLargeOutputCapture(t *testing.T) {
	e := NewExecutable("./test_helpers/large_echo.sh")
	result, err := e.Run("hey")

	assert.NoError(t, err)
	assert.Equal(t, 30000, len(result.Stdout))
	assert.Equal(t, "blah\n", string(result.Stderr))
}

func TestExitCode(t *testing.T) {
	e := NewExecutable("./test_helpers/exit_with.sh")

	result, _ := e.Run("0")
	assert.Equal(t, 0, result.ExitCode)

	result, _ = e.Run("1")
	assert.Equal(t, 1, result.ExitCode)

	result, _ = e.Run("2")
	assert.Equal(t, 2, result.ExitCode)
}

func TestExecutableStartNotAllowedIfInProgress(t *testing.T) {
	e := NewExecutable("./test_helpers/sleep_for.sh")

	// Run once
	err := e.Start("0.01")
	assert.NoError(t, err)

	// Starting again when in progress should throw an error
	err = e.Start("0.01")
	assertErrorContains(t, err, "process already in progress")

	// Running again when in progress should throw an error
	_, err = e.Run("0.01")
	assertErrorContains(t, err, "process already in progress")

	e.Wait()

	// Running again once finished should be fine
	err = e.Start("0.01")
	assert.NoError(t, err)
}

func TestSuccessiveExecutions(t *testing.T) {
	e := NewExecutable("./test_helpers/stdout_echo.sh")

	result, _ := e.Run("1")
	assert.Equal(t, "1\n", string(result.Stdout))

	result, _ = e.Run("2")
	assert.Equal(t, "2\n", string(result.Stdout))
}

func TestHasExited(t *testing.T) {
	e := NewExecutable("./test_helpers/sleep_for.sh")

	e.Start("0.1")
	assert.False(t, e.HasExited(), "Expected to not have exited")

	time.Sleep(150 * time.Millisecond)
	assert.True(t, e.HasExited(), "Expected to have exited")
}

func TestStdin(t *testing.T) {
	e := NewExecutable("grep")

	e.Start("cat")
	assert.False(t, e.HasExited(), "Expected to not have exited")

	e.stdioHandler.GetStdin().Write([]byte("has cat"))
	assert.False(t, e.HasExited(), "Expected to not have exited")

	e.stdioHandler.GetStdin().Close()
	time.Sleep(100 * time.Millisecond)
	assert.True(t, e.HasExited(), "Expected to have exited")
}

func TestRunWithStdin(t *testing.T) {
	e := NewExecutable("grep")

	result, err := e.RunWithStdin([]byte("has cat"), "cat")
	assert.NoError(t, err)

	assert.Equal(t, result.ExitCode, 0)

	result, err = e.RunWithStdin([]byte("only dog"), "cat")
	assert.NoError(t, err)

	assert.Equal(t, result.ExitCode, 1)
}

func TestRunWithStdinTimeout(t *testing.T) {
	e := NewExecutable("sleep")
	e.TimeoutInMilliseconds = 50

	result, err := e.RunWithStdin([]byte(""), "10")
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "execution timed out")

	result, err = e.RunWithStdin([]byte(""), "0.02")
	assert.NoError(t, err)
	assert.Equal(t, result.ExitCode, 0)
}

// Rogue == doesn't respond to SIGTERM
func TestTerminatesRoguePrograms(t *testing.T) {
	e := NewExecutable("bash")

	err := e.Start("-c", "trap '' SIGTERM SIGINT; sleep 60")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = e.Kill()
	assert.EqualError(t, err, "program failed to exit in 2 seconds after receiving sigterm")

	// Starting again shouldn't throw an error
	err = e.Start("-c", "trap '' SIGTERM SIGINT; sleep 60")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = e.Kill()
	assert.EqualError(t, err, "program failed to exit in 2 seconds after receiving sigterm")
}

func TestSegfault(t *testing.T) {
	e := NewExecutable("./test_helpers/segfault.sh")

	result, err := e.Run()
	assert.NoError(t, err)
	assert.Equal(t, 139, result.ExitCode)
}

func TestMemoryLimit(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Memory limiting is only supported on Linux")
	}

	e := NewExecutable("./test_helpers/memory_hog.sh")
	// Set a 50MB memory limit
	e.MemoryLimitInBytes = 50 * 1024 * 1024
	e.TimeoutInMilliseconds = 30 * 1000 // 30 seconds should be plenty

	result, err := e.Run()
	errorIsMemoryLimitExceeded := errors.Is(err, ErrMemoryLimitExceeded)
	errorStderrContainsMemoryError := strings.Contains(string(result.Stderr), "MemoryError")
	assert.True(t, errorIsMemoryLimitExceeded || errorStderrContainsMemoryError, `Expected MemoryLimitExceeded.
errorIsMemoryLimitExceeded: %t
errorStderrContainsMemoryError: %t
`, errorIsMemoryLimitExceeded, errorStderrContainsMemoryError)
	if err != nil {
		assert.Contains(t, err.Error(), "50 MB", "Error message should contain human-readable memory limit")
	}
}
