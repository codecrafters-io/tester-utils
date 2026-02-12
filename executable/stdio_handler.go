package executable

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type stdioHandler interface {
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

// pipeTrioStdioHandler deals with pipe based i/o
type pipeTrioStdioHandler struct {
	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
}

func (h *pipeTrioStdioHandler) GetStdin() io.WriteCloser {
	return h.stdinPipe
}

func (h *pipeTrioStdioHandler) GetStdout() io.ReadCloser {
	return h.stdoutPipe
}

func (h *pipeTrioStdioHandler) GetStderr() io.ReadCloser {
	return h.stderrPipe
}

func (h *pipeTrioStdioHandler) SetupStreams(cmd *exec.Cmd) error {
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

func (h *pipeTrioStdioHandler) CloseChildStreams() error {
	// No action needed here: closing child streams is automatically handled by exec library
	return nil
}

func (h *pipeTrioStdioHandler) CloseParentStreams() error {
	return closeAllWithCloserFunc(closeIfOpen, h.stdinPipe, h.stdoutPipe, h.stderrPipe)
}

func (h *pipeTrioStdioHandler) TerminateStdin() error {
	if err := h.stdinPipe.Close(); err != nil {
		return err
	}

	return nil
}

// pipeInPtysOutStdioHandler deals with PTY based i/o
// It uses a pipe for stdin and pty devices for stdout and stderr
type pipeInPtysOutStdioHandler struct {
	stdoutMaster, stdoutSlave *os.File
	stderrMaster, stderrSlave *os.File
	stdinPipe                 io.WriteCloser
}

func (h *pipeInPtysOutStdioHandler) GetStdin() io.WriteCloser {
	return h.stdinPipe
}

func (h *pipeInPtysOutStdioHandler) GetStdout() io.ReadCloser {
	return h.stdoutMaster
}

func (h *pipeInPtysOutStdioHandler) GetStderr() io.ReadCloser {
	return h.stderrMaster
}

func (h *pipeInPtysOutStdioHandler) SetupStreams(cmd *exec.Cmd) error {
	if err := h.openAll(); err != nil {
		return err
	}

	var err error
	h.stdinPipe, err = cmd.StdinPipe()

	if err != nil {
		return err
	}

	cmd.Stdout = h.stdoutSlave
	cmd.Stderr = h.stderrSlave

	return nil
}

func (h *pipeInPtysOutStdioHandler) CloseChildStreams() error {
	// Close slave ends - child process now owns them
	return h.closeSlaves()
}

func (h *pipeInPtysOutStdioHandler) CloseParentStreams() error {
	return h.closeMasters()
}

func (h *pipeInPtysOutStdioHandler) TerminateStdin() error {
	return h.stdinPipe.Close()
}

// openAll attempts to open all PTY pairs.
// Returns an error if any PTY fails to open, and automatically cleans up any successfully opened PTYs.
func (r *pipeInPtysOutStdioHandler) openAll() error {
	var err error

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
func (r *pipeInPtysOutStdioHandler) closeAll() error {
	var firstError error

	// best effort
	if closeMasterError := r.closeMasters(); closeMasterError != nil {
		firstError = closeMasterError
	}

	if closeSlaveError := r.closeSlaves(); closeSlaveError != nil && firstError == nil {
		firstError = closeSlaveError
	}

	return firstError
}

// closeSlaves closes only the slave ends of the PTY pairs.
func (r *pipeInPtysOutStdioHandler) closeSlaves() error {
	// PTY are managed by ptyStdioHandler alone, and are not modified externally, so
	// closeIfOpen() is not needed here
	return closeAllWithCloserFunc(closeIfNotNil, r.stdoutSlave, r.stderrSlave)
}

// closeMasters closes only the master ends of the PTY pairs.
func (r *pipeInPtysOutStdioHandler) closeMasters() error {
	return closeAllWithCloserFunc(closeIfNotNil, r.stdoutMaster, r.stderrMaster)
}
