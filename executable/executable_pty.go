package executable

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// ptyResources manages the three PTY pairs used for stdin, stdout, and stderr
type ptyResources struct {
	stdinMaster, stdinSlave   *os.File
	stdoutMaster, stdoutSlave *os.File
	stderrMaster, stderrSlave *os.File
}

// openAll attempts to open all three PTY pairs.
// Returns an error if any PTY fails to open, and automatically cleans up any successfully opened PTYs.
func (r *ptyResources) openAll() error {
	var err error

	r.stdinMaster, r.stdinSlave, err = pty.Open()
	if err != nil {
		return err
	}

	r.stdoutMaster, r.stdoutSlave, err = pty.Open()
	if err != nil {
		r.closeAll()
		return err
	}

	r.stderrMaster, r.stderrSlave, err = pty.Open()
	if err != nil {
		r.closeAll()
		return err
	}

	return nil
}

// closeAll closes all PTY file descriptors.
func (r *ptyResources) closeAll() {
	if r.stdinMaster != nil {
		r.stdinMaster.Close()
	}
	if r.stdinSlave != nil {
		r.stdinSlave.Close()
	}
	if r.stdoutMaster != nil {
		r.stdoutMaster.Close()
	}
	if r.stdoutSlave != nil {
		r.stdoutSlave.Close()
	}
	if r.stderrMaster != nil {
		r.stderrMaster.Close()
	}
	if r.stderrSlave != nil {
		r.stderrSlave.Close()
	}
}

// closeSlaves closes only the slave ends of the PTY pairs.
// This is called after cmd.Start() succeeds to remove references in the parent process.
func (r *ptyResources) closeSlaves() {
	if r.stdinSlave != nil {
		r.stdinSlave.Close()
	}
	if r.stdoutSlave != nil {
		r.stdoutSlave.Close()
	}
	if r.stderrSlave != nil {
		r.stderrSlave.Close()
	}
}

// StartInPty starts the specified command with PTY support but does not wait for it to complete.
func (e *Executable) StartInPty(args ...string) error {
	// Use three different PTY pairs
	// This removes input reflection checks: Anything written to stdin won't appear in stdout/stderr
	// Stdout and Stderr messages of a process are segregated, allowing a clear approach to
	// record a process's output streams
	var resources ptyResources

	if err := resources.openAll(); err != nil {
		return err
	}

	stdioInitializer := func(cmd *exec.Cmd) error {
		cmd.Stdin = resources.stdinSlave
		cmd.Stdout = resources.stdoutSlave
		cmd.Stderr = resources.stderrSlave

		e.stdinStream = resources.stdinMaster
		e.stdoutStream = resources.stdoutMaster
		e.stderrStream = resources.stderrMaster
		return nil
	}

	// Close the slave ends of the PTY pair
	// This is to remove any references to the slave pair in the parent process (tester)
	// So that when the process exits, all the streams will have been properly closed
	slaveCloserCallback := func() {
		resources.closeSlaves()
	}

	err := e.startWithCallbacks(stdioInitializer, slaveCloserCallback, args...)
	if err != nil {
		resources.closeAll()
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
