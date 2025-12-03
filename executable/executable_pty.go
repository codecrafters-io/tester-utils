package executable

import (
	"os/exec"

	"github.com/creack/pty"
)

// StartInPty starts the specified command with PTY support but does not wait for it to complete.
func (e *Executable) StartInPty(args ...string) error {
	// Use three different PTY pairs
	// This removes input reflection checks: Anything written to stdin won't appear in stdout/stderr
	// Stdout and Stderr messages of a process are segregated, allowing a clear approach to
	// record a process's output streams
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

	// Close the slave ends of the PTY pair
	// This is to remove any references to the slave pair in the parent process (tester)
	// So that when the process exits, all the streams will have been properly closed
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

	if _, err = e.stdinStream.Write(stdin); err != nil {
		return ExecutableResult{}, err
	}

	return e.Wait()
}
