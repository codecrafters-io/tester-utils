package executable

import (
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

// closeIfNotNil closes a file descriptor if it is not nil
func closeIfNotNil(f *os.File) error {
	if f != nil {
		return f.Close()
	}

	return nil
}
