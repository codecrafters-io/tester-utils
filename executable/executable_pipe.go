package executable

import "os/exec"

// Start starts the specified command but does not wait for it to complete.
func (e *Executable) Start(args ...string) error {
	stdStreamsInitializer := func(cmd *exec.Cmd) error {
		var err error

		e.stdoutStream, err = cmd.StdoutPipe()
		if err != nil {
			return err
		}

		e.stderrStream, err = cmd.StderrPipe()
		if err != nil {
			return err
		}

		e.stdinStream, err = cmd.StdinPipe()
		if err != nil {
			return err
		}

		return nil
	}

	return e.startWithCallbacks(stdStreamsInitializer, nil, args...)
}

// Run starts the specified command, waits for it to complete and returns the result.
func (e *Executable) Run(args ...string) (ExecutableResult, error) {
	var err error

	if err = e.Start(args...); err != nil {
		return ExecutableResult{}, err
	}

	return e.Wait()
}

// RunWithStdin starts the specified command, sends input, waits for it to complete and returns the result.
func (e *Executable) RunWithStdin(stdin []byte, args ...string) (ExecutableResult, error) {
	var err error

	if err = e.Start(args...); err != nil {
		return ExecutableResult{}, err
	}

	e.stdinStream.Write(stdin)

	return e.Wait()
}
