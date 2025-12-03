package executable

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStartInPty(t *testing.T) {
	err := NewExecutable("/blah").StartInPty()
	assertErrorContains(t, err, "not found")
	assertErrorContains(t, err, "blah")

	err = NewExecutable("./test_helpers/not_executable.sh").StartInPty()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "not_executable.sh")

	err = NewExecutable("./test_helpers/haskell").StartInPty()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "haskell")

	err = NewExecutable("./test_helpers/stdout_echo.sh").StartInPty()
	assert.NoError(t, err)
}

func TestStartAndKillInPty(t *testing.T) {
	e := NewExecutable("/blah")
	err := e.StartInPty()
	assertErrorContains(t, err, "not found")
	assertErrorContains(t, err, "blah")
	err = e.Kill()
	assert.NoError(t, err)

	e = NewExecutable("./test_helpers/not_executable.sh")
	err = e.StartInPty()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "not_executable.sh")
	err = e.Kill()
	assert.NoError(t, err)

	e = NewExecutable("./test_helpers/haskell")
	err = e.StartInPty()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "haskell")
	err = e.Kill()
	assert.NoError(t, err)

	e = NewExecutable("./test_helpers/stdout_echo.sh")
	err = e.StartInPty()
	assert.NoError(t, err)
	err = e.Kill()
	assert.NoError(t, err)
}

func TestRunInPty(t *testing.T) {
	e := NewExecutable("./test_helpers/stdout_echo.sh")
	result, err := e.RunWithStdinInPty([]byte(""), "hey")
	assert.NoError(t, err)
	assert.Equal(t, "hey\r\n", string(result.Stdout))
}

func TestOutputCaptureInPty(t *testing.T) {
	// Stdout capture
	e := NewExecutable("./test_helpers/stdout_echo.sh")
	result, err := e.RunWithStdinInPty([]byte(""), "hey")

	assert.NoError(t, err)
	assert.Equal(t, "hey\r\n", string(result.Stdout))
	assert.Equal(t, "", string(result.Stderr))

	// Stderr capture
	e = NewExecutable("./test_helpers/stderr_echo.sh")
	result, err = e.RunWithStdinInPty([]byte(""), "hey")

	assert.NoError(t, err)
	assert.Equal(t, "", string(result.Stdout))
	assert.Equal(t, "hey\r\n", string(result.Stderr))
}

func TestLargeOutputCaptureInPty(t *testing.T) {
	e := NewExecutable("./test_helpers/large_echo.sh")
	result, err := e.RunWithStdinInPty([]byte(""), "hey")

	assert.NoError(t, err)
	assert.Equal(t, 30000, len(result.Stdout))
	assert.Equal(t, "blah\r\n", string(result.Stderr))
}

func TestExitCodeInPty(t *testing.T) {
	e := NewExecutable("./test_helpers/exit_with.sh")
	e.TimeoutInMilliseconds = 3600 * 1000

	result, _ := e.RunWithStdinInPty([]byte(""), "0")
	assert.Equal(t, 0, result.ExitCode)

	result, _ = e.RunWithStdinInPty([]byte(""), "1")
	assert.Equal(t, 1, result.ExitCode)

	result, _ = e.RunWithStdinInPty([]byte(""), "2")
	assert.Equal(t, 2, result.ExitCode)
}

func TestExecutableStartNotAllowedIfInProgressInPty(t *testing.T) {
	e := NewExecutable("./test_helpers/sleep_for.sh")

	// Run once
	err := e.StartInPty("0.01")
	assert.NoError(t, err)

	// Starting again when in progress should throw an error
	err = e.StartInPty("0.01")
	assertErrorContains(t, err, "process already in progress")

	// Running again when in progress should throw an error
	_, err = e.RunWithStdinInPty([]byte(""), "0.01")
	assertErrorContains(t, err, "process already in progress")

	e.Wait()

	// Running again once finished should be fine
	err = e.StartInPty("0.01")
	assert.NoError(t, err)
}

func TestSuccessiveExecutionsInPty(t *testing.T) {
	e := NewExecutable("./test_helpers/stdout_echo.sh")

	result, _ := e.RunWithStdinInPty([]byte(""), "1")
	assert.Equal(t, "1\r\n", string(result.Stdout))

	result, _ = e.RunWithStdinInPty([]byte(""), "2")
	assert.Equal(t, "2\r\n", string(result.Stdout))
}

func TestHasExitedInPty(t *testing.T) {
	e := NewExecutable("./test_helpers/sleep_for.sh")

	e.StartInPty("0.1")
	assert.False(t, e.HasExited(), "Expected to not have exited")

	time.Sleep(150 * time.Millisecond)
	assert.True(t, e.HasExited(), "Expected to have exited")
}

func TestStdinInPty(t *testing.T) {
	e := NewExecutable("grep")

	e.StartInPty("cat")
	assert.False(t, e.HasExited(), "Expected to not have exited")

	e.stdinStream.Write([]byte("has cat"))
	assert.False(t, e.HasExited(), "Expected to not have exited")

	e.stdinStream.Close()
	time.Sleep(100 * time.Millisecond)
	assert.True(t, e.HasExited(), "Expected to have exited")
}

func TestRunWithStdinInPty(t *testing.T) {
	e := NewExecutable("grep")

	result, err := e.RunWithStdinInPty([]byte("has cat\n"), "cat")
	assert.NoError(t, err)

	assert.Equal(t, result.ExitCode, 0)

	result, err = e.RunWithStdinInPty([]byte("only dog\n"), "cat")
	assert.NoError(t, err)

	assert.Equal(t, result.ExitCode, 1)
}

func TestRunWithStdinTimeoutInPty(t *testing.T) {
	e := NewExecutable("sleep")
	e.TimeoutInMilliseconds = 50

	result, err := e.RunWithStdinInPty([]byte(""), "10")
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "execution timed out")

	result, err = e.RunWithStdinInPty([]byte(""), "0.01") // Reduced sleep time to 10ms
	assert.NoError(t, err)
	assert.Equal(t, result.ExitCode, 0)
}

// Rogue == doesn't respond to SIGTERM
func TestTerminatesRogueProgramsInPty(t *testing.T) {
	e := NewExecutable("bash")

	err := e.StartInPty("-c", "trap '' SIGTERM SIGINT; sleep 60")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = e.Kill()
	assert.EqualError(t, err, "program failed to exit in 2 seconds after receiving sigterm")

	// Starting again shouldn't throw an error
	err = e.StartInPty("-c", "trap '' SIGTERM SIGINT; sleep 60")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = e.Kill()
	assert.EqualError(t, err, "program failed to exit in 2 seconds after receiving sigterm")
}

func TestSegfaultInPty(t *testing.T) {
	e := NewExecutable("./test_helpers/segfault.sh")

	result, err := e.RunWithStdinInPty([]byte(""), "")
	assert.NoError(t, err)
	assert.Equal(t, 139, result.ExitCode)
}
