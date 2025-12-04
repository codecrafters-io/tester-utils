package executable

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type streamOwner int

const (
	child streamOwner = iota
	parent
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

	// (TODO|REMOVE) I'll remove this after the PR review.
	// I thought about simplifiying this further
	// This could just be CloseStreams() and we only deal with closing parent's end of the FD's here, and let SetupStreams() close the child's end of the streams
	// However, the child's end of the FD's need to be closed AFTER cmd.Start(), but the SetupStreams() is run before cmd.Start()
	// We could, of course, make SetupStreams() return a callback function to clean up child's FDs, but using callbacks just feels like regressing back to the callback-style pattern

	// CloseStreams closes file descriptors based on stream owner
	// Child: closes FDs duplicated for child (called after cmd.Start())
	// Parent: closes parent's remaining FDs (called during cleanup)
	CloseStreams(owner streamOwner) error

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
		return err
	}

	if h.stderrPipe, err = cmd.StderrPipe(); err != nil {
		return err
	}

	return nil
}

func (h *pipeStdioHandler) CloseStreams(owner streamOwner) error {
	switch owner {
	case child:
		// No action needed here: closing child streams is automatically handled by exec library
		return nil
	case parent:
		return h.closeParentStreams()
	default:
		panic(fmt.Sprintf("Codecrafters Internal Error - Wrong owner type in pipeStdioHandler.CloseStreams(): %v", owner))
	}
}

func (h *pipeStdioHandler) TerminateStdin() error {
	if err := h.stdinPipe.Close(); err != nil {
		return err
	}

	return nil
}

func (h *pipeStdioHandler) closeParentStreams() error {
	// In case of pipes, closing of parent streams may be automatically handled by the exec library in certain cases
	// For eg. if cmd.Start() fails, or after cmd.Wait() is run
	// So, we close the parent streams only if they're not already closed
	return closeAllWithCloserFunc(closeIfOpen, h.stdinPipe, h.stdoutPipe, h.stderrPipe)
}

// ptyStdioHandler deals with PTY based i/o
type ptyStdioHandler struct {
	stdoutMaster, stdoutSlave *os.File
	stderrMaster, stderrSlave *os.File
	stdinMaster, stdinSlave   *os.File
}

func (h *ptyStdioHandler) GetStdin() io.WriteCloser {
	return h.stdinMaster
}

func (h *ptyStdioHandler) GetStdout() io.ReadCloser {
	return h.stdoutMaster
}

func (h *ptyStdioHandler) GetStderr() io.ReadCloser {
	return h.stderrMaster
}

func (h *ptyStdioHandler) SetupStreams(cmd *exec.Cmd) error {
	if err := h.openAll(); err != nil {
		return err
	}

	// Assign slave end of PTYs to the child process
	cmd.Stdin = h.stdinSlave
	cmd.Stdout = h.stdoutSlave
	cmd.Stderr = h.stderrSlave

	return nil
}

func (h *ptyStdioHandler) CloseStreams(owner streamOwner) error {
	switch owner {
	case child:
		// Close slave ends - child process now owns them
		return h.closeSlaves()
	case parent:
		// Close master ends - parent cleanup
		return h.closeMasters()
	default:
		panic(fmt.Sprintf("Codecrafters Internal Error - Wrong owner type in ptyStdioHandler.CloseStreams(): %v", owner))
	}
}

func (h *ptyStdioHandler) TerminateStdin() error {
	// Send (\n + Ctrl-D) for closing input stream
	_, err := h.stdinMaster.Write([]byte("\n\004"))
	return err
}

// openAll attempts to open all three PTY pairs.
// Returns an error if any PTY fails to open, and automatically cleans up any successfully opened PTYs.
func (r *ptyStdioHandler) openAll() error {
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
func (r *ptyStdioHandler) closeAll() error {
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
func (r *ptyStdioHandler) closeSlaves() error {
	// PTY are managed by ptyStdioHandler alone, and are not modified externally, so
	// closeIfOpen() is not needed here
	return closeAllWithCloserFunc(closeIfNotNil, r.stdinSlave, r.stdoutSlave, r.stderrSlave)
}

// closeMasters closes only the master ends of the PTY pairs.
func (r *ptyStdioHandler) closeMasters() error {
	return closeAllWithCloserFunc(closeIfNotNil, r.stdinMaster, r.stdoutMaster, r.stderrMaster)
}
