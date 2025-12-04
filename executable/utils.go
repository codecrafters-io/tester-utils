package executable

import (
	"errors"
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

// isTTY returns true if the object is a tty
func isTTY(o any) bool {
	file, ok := o.(*os.File)
	if !ok {
		return false
	}

	return isatty.IsTerminal(file.Fd())
}

// closeIfNotNil closes an io.Closer if it is not already closed
func closeIfNotNil(c io.Closer) error {
	if c != nil {
		return c.Close()
	}

	return nil
}

// closeIfOpen closes an io.Closer if it is not already closed
func closeIfOpen(c io.Closer) error {
	if c == nil {
		return nil
	}

	err := c.Close()
	if err != nil && !errors.Is(err, os.ErrClosed) {
		return err
	}

	return nil
}

// closeStdStreamsUsingCloserFunction attempts to close stdin, stdout, and stderr using the provided close function.
// Returns the first error encountered, but continues attempting to close all streams.
func closeStdStreamsUsingCloserFunction(closeFunc func(io.Closer) error, stdin, stdout, stderr io.Closer) error {
	var firstError error

	if stdinError := closeFunc(stdin); stdinError != nil {
		firstError = stdinError
	}

	if stdoutError := closeFunc(stdout); stdoutError != nil && firstError == nil {
		firstError = stdoutError
	}

	if stderrError := closeFunc(stderr); stderrError != nil && firstError == nil {
		firstError = stderrError
	}

	return firstError
}
