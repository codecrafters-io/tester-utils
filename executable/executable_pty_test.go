package executable

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// createPTYOptions creates PTYOptions with stdout using PTY (most common case)
func createPTYOptions() (*PTYOptions, error) {
	return &PTYOptions{
		UsePipeForStdin:  false, // Use PTY
		UsePipeForStdout: false, // Use PTY
		UsePipeForStderr: false, // Use PTY
	}, nil
}

// runInPTY is a helper method to run an executable with PTY support
func (e *Executable) runInPTY(ptyOptions *PTYOptions, args ...string) (ExecutableResult, error) {
	if err := e.StartInPty(ptyOptions, args...); err != nil {
		return ExecutableResult{}, err
	}
	return e.Wait()
}

// PTY Test Cases - Mirror of all existing tests but using PTY

func TestStartInPTY(t *testing.T) {
	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	err = NewExecutable("/blah").StartInPty(ptyOptions)
	assertErrorContains(t, err, "not found")
	assertErrorContains(t, err, "blah")

	ptyOptions2, err := createPTYOptions()
	assert.NoError(t, err)

	err = NewExecutable("./test_helpers/not_executable.sh").StartInPty(ptyOptions2)
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "not_executable.sh")

	ptyOptions3, err := createPTYOptions()
	assert.NoError(t, err)

	err = NewExecutable("./test_helpers/haskell").StartInPty(ptyOptions3)
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "haskell")

	ptyOptions4, err := createPTYOptions()
	assert.NoError(t, err)

	e := NewExecutable("./test_helpers/stdout_echo.sh")
	err = e.StartInPty(ptyOptions4)
	assert.NoError(t, err)
}

func TestStartAndKillInPTY(t *testing.T) {
	e1 := NewExecutable("/blah")
	ptyOptions1, err := createPTYOptions()
	assert.NoError(t, err)

	err = e1.StartInPty(ptyOptions1)
	assertErrorContains(t, err, "not found")
	assertErrorContains(t, err, "blah")

	e2 := NewExecutable("./test_helpers/not_executable.sh")
	ptyOptions2, err := createPTYOptions()
	assert.NoError(t, err)

	err = e2.StartInPty(ptyOptions2)
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "not_executable.sh")

	e3 := NewExecutable("./test_helpers/haskell")
	ptyOptions3, err := createPTYOptions()
	assert.NoError(t, err)

	err = e3.StartInPty(ptyOptions3)
	assertErrorContains(t, err, "not an executable file")
	assertErrorContains(t, err, "haskell")

	e4 := NewExecutable("./test_helpers/stdout_echo.sh")
	ptyOptions4, err := createPTYOptions()
	assert.NoError(t, err)

	err = e4.StartInPty(ptyOptions4, "test")
	assert.NoError(t, err)

	result, err := e4.Wait()
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestKillInPTY(t *testing.T) {
	e := NewExecutable("./test_helpers/sleep_for.sh")
	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	err = e.StartInPty(ptyOptions, "0.5")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = e.Kill()
	assert.NoError(t, err)
}

func TestRunInPTY(t *testing.T) {
	e := NewExecutable("./test_helpers/stdout_echo.sh")
	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	result, err := e.runInPTY(ptyOptions, "hey")
	assert.NoError(t, err)
	assert.Equal(t, "hey\r\n", string(result.Stdout))
}

func TestOutputCaptureInPTY(t *testing.T) {
	e := NewExecutable("./test_helpers/stdout_echo.sh")
	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	result, err := e.runInPTY(ptyOptions, "hey")
	assert.NoError(t, err)
	assert.Equal(t, "hey\r\n", string(result.Stdout))
	assert.Equal(t, "", string(result.Stderr))

	e = NewExecutable("./test_helpers/stderr_echo.sh")
	ptyOptions2, err := createPTYOptions()
	assert.NoError(t, err)

	result, err = e.runInPTY(ptyOptions2, "hey")
	assert.NoError(t, err)
	assert.Equal(t, "", string(result.Stdout))
	assert.Equal(t, "hey\n", string(result.Stderr))
}

func TestLargeOutputCaptureInPTY(t *testing.T) {
	e := NewExecutable("./test_helpers/large_echo.sh")
	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	result, err := e.runInPTY(ptyOptions, "hey")
	assert.NoError(t, err)
	assert.Equal(t, 30000, len(result.Stdout))
	assert.Equal(t, "blah\n", string(result.Stderr))
}

func TestExitCodeInPTY(t *testing.T) {
	e := NewExecutable("./test_helpers/exit_with.sh")

	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	result, _ := e.runInPTY(ptyOptions, "0")
	assert.Equal(t, 0, result.ExitCode)

	ptyOptions2, err := createPTYOptions()
	assert.NoError(t, err)

	result, _ = e.runInPTY(ptyOptions2, "1")
	assert.Equal(t, 1, result.ExitCode)

	ptyOptions3, err := createPTYOptions()
	assert.NoError(t, err)

	result, _ = e.runInPTY(ptyOptions3, "2")
	assert.Equal(t, 2, result.ExitCode)
}

func TestExecutableStartInPTYNotAllowedIfInProgress(t *testing.T) {
	e := NewExecutable("./test_helpers/sleep_for.sh")

	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	err = e.StartInPty(ptyOptions, "0.01")
	assert.NoError(t, err)

	ptyOptions2, err := createPTYOptions()
	assert.NoError(t, err)

	err = e.StartInPty(ptyOptions2, "0.01")
	assertErrorContains(t, err, "process already in progress")

	ptyOptions3, err := createPTYOptions()
	assert.NoError(t, err)

	_, err = e.runInPTY(ptyOptions3, "0.01")
	assertErrorContains(t, err, "process already in progress")

	e.Wait()

	ptyOptions4, err := createPTYOptions()
	assert.NoError(t, err)

	err = e.StartInPty(ptyOptions4, "0.01")
	assert.NoError(t, err)
}

func TestSuccessiveExecutionsInPTY(t *testing.T) {
	e := NewExecutable("./test_helpers/stdout_echo.sh")

	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	result, _ := e.runInPTY(ptyOptions, "1")
	assert.Equal(t, "1\r\n", string(result.Stdout))

	ptyOptions2, err := createPTYOptions()
	assert.NoError(t, err)

	result, _ = e.runInPTY(ptyOptions2, "2")
	assert.Equal(t, "2\r\n", string(result.Stdout))
}

func TestHasExitedInPTY(t *testing.T) {
	e := NewExecutable("./test_helpers/sleep_for.sh")
	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	e.StartInPty(ptyOptions, "0.1")
	assert.False(t, e.HasExited())

	time.Sleep(150 * time.Millisecond)
	assert.True(t, e.HasExited())
}

func TestStdinInPTY(t *testing.T) {
	e := NewExecutable("grep")

	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	e.StartInPty(ptyOptions, "cat")
	assert.False(t, e.HasExited())

	e.StdinPipe.Write([]byte("has cat"))
	assert.False(t, e.HasExited())

	e.StdinPipe.Close()
	time.Sleep(100 * time.Millisecond)
	assert.True(t, e.HasExited())
}

func TestRunWithStdinInPTY(t *testing.T) {
	e := NewExecutable("grep")

	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	err = e.StartInPty(ptyOptions, "cat")
	assert.NoError(t, err)

	e.StdinPipe.Write([]byte("has cat"))
	result, err := e.Wait()
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)

	e = NewExecutable("grep")
	ptyOptions2, err := createPTYOptions()
	assert.NoError(t, err)

	err = e.StartInPty(ptyOptions2, "cat")
	assert.NoError(t, err)

	e.StdinPipe.Write([]byte("only dog"))
	result, err = e.Wait()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.ExitCode)
}

func TestRunWithStdinTimeoutInPTY(t *testing.T) {
	e := NewExecutable("sleep")
	e.TimeoutInMilliseconds = 50

	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	result, err := e.runInPTY(ptyOptions, "10")
	assert.Error(t, err)
	assert.Equal(t, "execution timed out", err.Error())

	ptyOptions2, err := createPTYOptions()
	assert.NoError(t, err)

	result, err = e.runInPTY(ptyOptions2, "0.02")
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestTerminatesRogueProgramsInPTY(t *testing.T) {
	e := NewExecutable("bash")

	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	err = e.StartInPty(ptyOptions, "-c", "trap '' SIGTERM SIGINT; sleep 60")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = e.Kill()
	assert.EqualError(t, err, "program failed to exit in 2 seconds after receiving sigterm")

	ptyOptions2, err := createPTYOptions()
	assert.NoError(t, err)

	err = e.StartInPty(ptyOptions2, "-c", "trap '' SIGTERM SIGINT; sleep 60")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = e.Kill()
	assert.EqualError(t, err, "program failed to exit in 2 seconds after receiving sigterm")
}

func TestSegfaultInPTY(t *testing.T) {
	e := NewExecutable("./test_helpers/segfault.sh")

	ptyOptions, err := createPTYOptions()
	assert.NoError(t, err)

	result, err := e.runInPTY(ptyOptions)
	assert.NoError(t, err)
	assert.Equal(t, 139, result.ExitCode)
}

func TestStartInPTYPanicCondition(t *testing.T) {
	e := NewExecutable("./test_helpers/stdout_echo.sh")

	masterR, slaveW, err := os.Pipe()
	assert.NoError(t, err)
	defer masterR.Close()
	defer slaveW.Close()

	ptyOptions := &PTYOptions{

		UsePipeForStdin:  true,
		UsePipeForStdout: true,
		UsePipeForStderr: true,
	}

	defer func() {
		if r := recover(); r != nil {
			assert.Contains(t, r.(string), "StartInTTY called with UsePipe for all three streams")
		} else {
			t.Error("Expected panic but didn't occur")
		}
	}()

	e.StartInPty(ptyOptions, "test")
}
