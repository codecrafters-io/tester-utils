package executable

import (
	"os"

	"github.com/mattn/go-isatty"
)

func isTTY(r any) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}

	return isatty.IsTerminal(file.Fd())
}
