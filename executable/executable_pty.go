package executable

import (
	"os/exec"

	"github.com/creack/pty"
)

// StartInPty starts the specified command with PTY support but does not wait for it to complete.
// The PTYOptions specify which streams should use the PTY vs regular pipes.
func (e *Executable) StartInPty(args ...string) error {
	// Use three different PTY pairs
	// This removes input reflection checks, and seggregates stdout and stderr messages of a process
	stdinMaster, stdinSlave, err := pty.Open()
	if err != nil {
		return err
	}

	stdoutMaster, stdoutSlave, err := pty.Open()
	if err != nil {
		return err
	}

	stderrMaster, stderrSlave, err := pty.Open()
	if err != nil {
		return err
	}

	stdioInitializer := func(cmd *exec.Cmd) error {
		cmd.Stdin = stdinSlave
		cmd.Stdout = stdoutSlave
		cmd.Stderr = stderrSlave

		e.stdinPipe = stdinMaster
		e.stdoutPipe = stdoutMaster
		e.stderrPipe = stderrMaster
		return nil
	}

	postStartHook := func() {
		stdinSlave.Close()
		stdoutSlave.Close()
		stderrSlave.Close()
	}

	return e.startWithHooks(stdioInitializer, postStartHook, args...)
}

// RunWithStdin starts the specified command, sends input, waits for it to complete and returns the
// result.
func (e *Executable) RunWithStdinInCLI(stdin []byte, args ...string) (ExecutableResult, error) {
	var err error

	if err = e.StartInPty(args...); err != nil {
		return ExecutableResult{}, err
	}

	e.stdinPipe.Write(stdin)

	return e.Wait()
}
