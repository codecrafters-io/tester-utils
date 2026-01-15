package executable

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"

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
	v := reflect.ValueOf(c)

	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil
	}

	return c.Close()
}

// closeIfOpen closes an io.Closer if it is not already closed
func closeIfOpen(c io.Closer) error {
	err := c.Close()

	if err != nil && !errors.Is(err, os.ErrClosed) {
		return err
	}

	return nil
}

// closeAllWithCloserFunc makes best effort (attempts to close all even in case of error)
// to close all the io.Closer interfacs using the provided closer function.
func closeAllWithCloserFunc(closer func(io.Closer) error, streams ...io.Closer) error {
	var firstError error
	for _, stream := range streams {
		if err := closer(stream); err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func resolveAbsolutePath(path string) (absolutePath string, err error) {
	// Try LookPath first
	absolutePath, err = exec.LookPath(path)

	if err != nil {
		// exec.LookPath returns error even if path is found.
		// When executable permissions and isDir() checks fail
		// We are only concerned with the path here
		absolutePath, err = filepath.Abs(path)

		if err != nil {
			return "", err
		}
	}

	return filepath.Abs(absolutePath)
}
