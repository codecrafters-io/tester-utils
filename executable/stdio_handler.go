package executable

import (
	"bytes"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type ExecutableStdioHandler interface {
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
}

// pipeStdioHandler deals with pipe based i/o
type pipeStdioHandler struct {
	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
}

func (h *pipeStdioHandler) GetStdin() io.WriteCloser {
	return h.stdinPipe
}

func (h *pipeStdioHandler) GetStdout() io.ReadCloser {
	return h.stdoutPipe
}

func (h *pipeStdioHandler) GetStderr() io.ReadCloser {
	return h.stderrPipe
}

func (h *pipeStdioHandler) SetupStreams(cmd *exec.Cmd) error {
	var err error

	if h.stdinPipe, err = cmd.StdinPipe(); err != nil {
		return err
	}

	if h.stdoutPipe, err = cmd.StdoutPipe(); err != nil {
		h.stdinPipe.Close()
		return err
	}

	if h.stderrPipe, err = cmd.StderrPipe(); err != nil {
		h.stdinPipe.Close()
		h.stdoutPipe.Close()
		return err
	}

	return nil
}

func (h *pipeStdioHandler) CloseChildStreams() error {
	// No action needed here: closing child streams is automatically handled by exec library
	return nil
}

func (h *pipeStdioHandler) CloseParentStreams() error {
	return closeAllWithCloserFunc(closeIfOpen, h.stdinPipe, h.stdoutPipe, h.stderrPipe)
}

func (h *pipeStdioHandler) TerminateStdin() error {
	if err := h.stdinPipe.Close(); err != nil {
		return err
	}

	return nil
}

// ptyStdioHandler deals with PTY based i/o
type ptyStdioHandler struct {
	master, slave *os.File
}

func (h *ptyStdioHandler) GetStdin() io.WriteCloser {
	return h.master
}

func (h *ptyStdioHandler) GetStdout() io.ReadCloser {
	return h.master
}

// GetStderr returns nothing
// expt: let's set a standard, Stderr is useless in case of pty spawn
func (h *ptyStdioHandler) GetStderr() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(nil))
}

func (h *ptyStdioHandler) SetupStreams(cmd *exec.Cmd) error {
	if err := h.openAll(); err != nil {
		return err
	}

	// Assign slave end of PTYs to the child process
	cmd.Stdin = h.slave
	cmd.Stdout = h.slave
	cmd.Stderr = h.slave

	return nil
}

func (h *ptyStdioHandler) CloseChildStreams() error {
	// Close slave ends - child process now owns them
	return h.closeSlaves()
}

func (h *ptyStdioHandler) CloseParentStreams() error {
	return h.closeMasters()
}

func (h *ptyStdioHandler) TerminateStdin() error {
	// Send (\n + Ctrl-D) for closing input stream
	_, err := h.master.Write([]byte("\n\004"))
	return err
}

// openAll attempts to open all three PTY pairs.
// Returns an error if any PTY fails to open, and automatically cleans up any successfully opened PTYs.
func (r *ptyStdioHandler) openAll() error {
	var err error

	r.master, r.slave, err = pty.Open()
	if err != nil {
		return err
	}

	return nil
}

// closeSlaves closes only the slave ends of the PTY pairs.
func (r *ptyStdioHandler) closeSlaves() error {
	// PTY are managed by ptyStdioHandler alone, and are not modified externally, so
	// closeIfOpen() is not needed here
	return closeAllWithCloserFunc(closeIfNotNil, r.slave)
}

// closeMasters closes only the master ends of the PTY pairs.
func (r *ptyStdioHandler) closeMasters() error {
	return closeAllWithCloserFunc(closeIfNotNil, r.master)
}
