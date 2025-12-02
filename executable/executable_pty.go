package executable

import (
	"os/exec"

	"github.com/creack/pty"
)

// StartInPty starts the specified command with PTY support but does not wait for it to complete.
// The PTYOptions specify which streams should use the PTY vs regular pipes.
func (e *Executable) StartInPty(ptyOptions *PTYOptions, args ...string) error {
	if ptyOptions.UsePipeForStdin && ptyOptions.UsePipeForStdout && ptyOptions.UsePipeForStderr {
		panic("Codecrafters Internal Error - StartInTTY called with UsePipe for all three streams")
	}

	master, slave, err := pty.Open()
	if err != nil {
		return err
	}

	stdioInitializer := func(cmd *exec.Cmd) error {
		var err error

		// Setup stdout
		if ptyOptions.UsePipeForStdout {
			e.stdoutPipe, err = cmd.StdoutPipe()
			if err != nil {
				return err
			}
		} else {
			cmd.Stdout = slave
			e.stdoutPipe = master
		}

		// Setup stderr
		if ptyOptions.UsePipeForStderr {
			e.stderrPipe, err = cmd.StderrPipe()
			if err != nil {
				return err
			}
		} else {
			cmd.Stderr = slave
			e.stderrPipe = master
		}

		// Setup stdin
		if ptyOptions.UsePipeForStdin {
			e.StdinPipe, err = cmd.StdinPipe()
			if err != nil {
				return err
			}
		} else {
			cmd.Stdin = slave
			e.StdinPipe = master
		}

		e.ptyMaster = master
		e.ptySlave = slave
		return nil
	}

	return e.startWithStdioInitializer(stdioInitializer, args...)
}

// RunInPty
func (e *Executable) RunInPty(ptyOptions PTYOptions, args ...string) (ExecutableResult, error) {
	return e.RunInPtyWithStdin(ptyOptions, nil, args...)
}

// RunInPtyWithStdin
func (e *Executable) RunInPtyWithStdin(ptyOptions PTYOptions, stdin []byte, args ...string) (ExecutableResult, error) {
	var err error

	if err = e.StartInPty(&ptyOptions, args...); err != nil {
		return ExecutableResult{}, err
	}

	if stdin != nil {
		e.ptyMaster.Write(stdin)
	}

	return e.Wait()
}
