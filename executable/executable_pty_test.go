package executable

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getNewExecutableForPTYTests(path string) *Executable {
	e := NewExecutable(path)
	e.ShouldusePTY = true
	return e
}

func TestStartInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("/blah")
	err := e.Start()

	assertErrorContains(t, err, "not found")
	assertErrorContains(t, err, "blah")

	err = getNewExecutableForPTYTests("./test_helpers/not_executable.sh").Start()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "not_executable.sh")

	err = getNewExecutableForPTYTests("./test_helpers/haskell").Start()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "haskell")

	err = getNewExecutableForPTYTests("./test_helpers/stdout_echo.sh").Start()
	assert.NoError(t, err)
}

func TestStartAndKillInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("/blah")

	err := e.Start()
	assertErrorContains(t, err, "not found")
	assertErrorContains(t, err, "blah")
	err = e.Kill()
	assert.NoError(t, err)

	e = getNewExecutableForPTYTests("./test_helpers/not_executable.sh")
	err = e.Start()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "not_executable.sh")
	err = e.Kill()
	assert.NoError(t, err)

	e = getNewExecutableForPTYTests("./test_helpers/haskell")
	err = e.Start()
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "haskell")
	err = e.Kill()
	assert.NoError(t, err)

	e = getNewExecutableForPTYTests("./test_helpers/stdout_echo.sh")
	err = e.Start()
	assert.NoError(t, err)
	err = e.Kill()
	assert.NoError(t, err)
}

func TestRunInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("./test_helpers/stdout_echo.sh")
	result, err := e.RunWithStdin([]byte(""), "hey")
	assert.NoError(t, err)
	assert.Equal(t, "hey\r\n", string(result.Stdout))
}

func TestOutputCaptureInPty(t *testing.T) {
	// Stdout capture
	e := getNewExecutableForPTYTests("./test_helpers/stdout_echo.sh")
	result, err := e.RunWithStdin([]byte(""), "hey")

	assert.NoError(t, err)
	assert.Equal(t, "hey\r\n", string(result.Stdout))
	assert.Equal(t, "", string(result.Stderr))

	// Stderr capture
	e = getNewExecutableForPTYTests("./test_helpers/stderr_echo.sh")
	result, err = e.RunWithStdin([]byte(""), "hey")

	assert.NoError(t, err)
	assert.Equal(t, "", string(result.Stdout))
	assert.Equal(t, "hey\r\n", string(result.Stderr))
}

func TestLargeOutputCaptureInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("./test_helpers/large_echo.sh")
	result, err := e.RunWithStdin([]byte(""), "hey")

	assert.NoError(t, err)
	assert.Equal(t, 30000, len(result.Stdout))
	assert.Equal(t, "blah\r\n", string(result.Stderr))
}

func TestExitCodeInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("./test_helpers/exit_with.sh")
	e.TimeoutInMilliseconds = 3600 * 1000

	result, _ := e.RunWithStdin([]byte(""), "0")
	assert.Equal(t, 0, result.ExitCode)

	result, _ = e.RunWithStdin([]byte(""), "1")
	assert.Equal(t, 1, result.ExitCode)

	result, _ = e.RunWithStdin([]byte(""), "2")
	assert.Equal(t, 2, result.ExitCode)
}

func TestExecutableStartNotAllowedIfInProgressInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("./test_helpers/sleep_for.sh")

	// Run once
	err := e.Start("0.01")
	assert.NoError(t, err)

	// Starting again when in progress should throw an error
	err = e.Start("0.01")
	assertErrorContains(t, err, "process already in progress")

	// Running again when in progress should throw an error
	_, err = e.RunWithStdin([]byte(""), "0.01")
	assertErrorContains(t, err, "process already in progress")

	e.Wait()

	// Running again once finished should be fine
	err = e.Start("0.01")
	assert.NoError(t, err)
}

func TestSuccessiveExecutionsInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("./test_helpers/stdout_echo.sh")

	result, _ := e.RunWithStdin([]byte(""), "1")
	assert.Equal(t, "1\r\n", string(result.Stdout))

	result, _ = e.RunWithStdin([]byte(""), "2")
	assert.Equal(t, "2\r\n", string(result.Stdout))
}

func TestHasExitedInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("./test_helpers/sleep_for.sh")

	e.Start("0.1")
	assert.False(t, e.HasExited(), "Expected to not have exited")

	time.Sleep(150 * time.Millisecond)
	assert.True(t, e.HasExited(), "Expected to have exited")
}

func TestStdinInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("grep")

	e.Start("cat")
	assert.False(t, e.HasExited(), "Expected to not have exited")

	e.stdinStream.Write([]byte("has cat"))
	assert.False(t, e.HasExited(), "Expected to not have exited")

	e.stdinStream.Close()
	time.Sleep(100 * time.Millisecond)
	assert.True(t, e.HasExited(), "Expected to have exited")
}

func TestRunWithStdinInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("grep")

	result, err := e.RunWithStdin([]byte("has cat"), "cat")
	assert.NoError(t, err)

	assert.Equal(t, result.ExitCode, 0)

	result, err = e.RunWithStdin([]byte("only dog"), "cat")
	assert.NoError(t, err)

	assert.Equal(t, result.ExitCode, 1)
}

func TestRunWithStdinTimeoutInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("sleep")
	// PTY incurs an overhead for setup and closure, so keeping the timeout larger than 50ms
	e.TimeoutInMilliseconds = 1000

	result, err := e.RunWithStdin([]byte(""), "10")
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "execution timed out")

	result, err = e.RunWithStdin([]byte(""), "0.01") // Reduced sleep time to 10ms
	assert.NoError(t, err)
	assert.Equal(t, result.ExitCode, 0)
}

// Rogue == doesn't respond to SIGTERM
func TestTerminatesRogueProgramsInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("bash")

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

func TestSegfaultInPty(t *testing.T) {
	e := getNewExecutableForPTYTests("./test_helpers/segfault.sh")

	result, err := e.RunWithStdin([]byte(""), "")
	assert.NoError(t, err)
	assert.Equal(t, 139, result.ExitCode)
}
