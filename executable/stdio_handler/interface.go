package stdio_handler

import (
	"io"
	"os/exec"
)

type StdioHandler interface {
	// GetStdin returns stdin on the parent's end
	GetStdin() io.WriteCloser

	// GetStdout returns stdout on the parent's end
	GetStdout() io.ReadCloser

	// GetStderr returns stderr on the parent's end
	GetStderr() io.ReadCloser

	// SetupStreams sets up child process' stdio streams
	SetupStreams(cmd *exec.Cmd) error

	// CloseChildStreams closes the FDs duplicated for child (called after cmd.Start())
	CloseChildStreams() error

	// CloseParentStreams() closes the FDs on the parent's end
	CloseParentStreams() error

	// TerminateStdin terminates the stdin interface of the child (effectively closes it)
	TerminateStdin() error

	Clone() StdioHandler
}
