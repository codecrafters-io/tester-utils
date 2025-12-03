package executable

import (
	"os/exec"

	"github.com/creack/pty"
)

// StartInPty starts the specified command with PTY support but does not wait for it to complete.
func (e *Executable) StartInPty(args ...string) error {
	// Use three different PTY pairs
	// This removes input reflection checks, and segregates stdout and stderr messages of a process
	stdinMaster, stdinSlave, err := pty.Open()
	if err != nil {
		return err
	}

	stdoutMaster, stdoutSlave, err := pty.Open()
	if err != nil {
		stdinMaster.Close()
		stdinSlave.Close()
		return err
	}

	stderrMaster, stderrSlave, err := pty.Open()
	if err != nil {
		stdinMaster.Close()
		stdinSlave.Close()
		stdoutMaster.Close()
		stdoutSlave.Close()
		return err
	}

	stdioInitializer := func(cmd *exec.Cmd) error {
		cmd.Stdin = stdinSlave
		cmd.Stdout = stdoutSlave
		cmd.Stderr = stderrSlave

		e.stdinStream = stdinMaster
		e.stdoutStream = stdoutMaster
		e.stderrStream = stderrMaster
		return nil
	}

	slaveCloserCallback := func() {
		stdinSlave.Close()
		stdoutSlave.Close()
		stderrSlave.Close()
	}

	err = e.startWithCallbacks(stdioInitializer, slaveCloserCallback, args...)

	if err != nil {
		stdinMaster.Close()
		stdoutMaster.Close()
		stderrMaster.Close()
		return err
	}

	return nil
}

// RunWithStdinInPty starts the specified command in a PTY, sends input, waits for it to complete and returns the result.
func (e *Executable) RunWithStdinInPty(stdin []byte, args ...string) (ExecutableResult, error) {
	var err error

	if err = e.StartInPty(args...); err != nil {
		return ExecutableResult{}, err
	}

	e.stdinStream.Write(stdin)

	return e.Wait()
}
