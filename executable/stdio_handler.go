package executable

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// (TO_DELETE) This comment will removed after PR review
// Went with uppercase(public) for interface methods, and used lowercase(private) for implementing
// struct's method just for visual differentiation
// Since this interface is private, it should not affect its usage

type stdioHandler interface {

	// getStdin(), getStdout(), and getStderr() are only used privately
	// these methods will allow exposing streams from executable in the future, if needed

	// GetStdin returns stdin on the parent's end
	GetStdin() io.WriteCloser

	// GetStdout returns stdout on the parent's end
	GetStdout() io.ReadCloser

	// GetStderr returns stderr on the parent's end
	GetStderr() io.ReadCloser

	// Sets up child process' stdio streams
	SetupStreams(cmd *exec.Cmd) error

	// closeStreamsOfChild closes the streams on the parent which were duplicated for the child's stdio streams
	CloseDuplicatedStreamsOfChild() error

	// cleanupOnFailedStart cleans up any FDs on the parent side if *exec.cmd.Start() fails
	CleanupStreamsOnFailedStart() error

	// CloseParentsEndOfChildStreams closes the parent's end of the child's stdio streams
	CloseParentsEndOfChildStreams() error

	// writeToStdinStream writes to child's stdin stream
	WriteToStdin(input []byte) (int, error)

	// Sends EOF to the child's stdin stream
	SendEofToStdin() error
}

// pipeStdioHandler deals with pipe based i/o
type pipeStdioHandler struct {
	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
	wasEofSent bool
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

func (h *pipeStdioHandler) CloseDuplicatedStreamsOfChild() error {
	// No implementation; this is automatically handled by *exec.Cmd.Start()
	return nil
}

func (h *pipeStdioHandler) CleanupStreamsOnFailedStart() error {
	if err := h.stdoutPipe.Close(); err != nil {
		return err
	}

	if err := h.stderrPipe.Close(); err != nil {
		return err
	}

	return h.stdinPipe.Close()
}

func (h *pipeStdioHandler) CloseParentsEndOfChildStreams() error {
	// No implementation; this is automatically handled by *exec.Cmd.Wait()
	return nil
}

func (h *pipeStdioHandler) WriteToStdin(input []byte) (int, error) {
	return h.stdinPipe.Write(input)
}

func (h *pipeStdioHandler) SendEofToStdin() error {
	if err := h.stdinPipe.Close(); err != nil {
		return err
	}

	h.wasEofSent = true
	return nil
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

func (h *ptyStdioHandler) CloseDuplicatedStreamsOfChild() error {
	return h.closeSlaves()
}

func (h *ptyStdioHandler) CleanupStreamsOnFailedStart() error {
	return h.closeMasters()
}

func (h *ptyStdioHandler) CloseParentsEndOfChildStreams() error {
	return h.closeMasters()
}

func (h *ptyStdioHandler) WriteToStdin(input []byte) (int, error) {
	// Terminal based input are only flushed after sending additional \n character
	return h.stdinMaster.Write(fmt.Appendf(input, "\n"))
}

func (h *ptyStdioHandler) SendEofToStdin() error {
	_, err := h.stdinMaster.Write([]byte{4})
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
	if err := r.closeMasters(); err != nil {
		return err
	}

	return r.closeSlaves()
}

// closeSlaves closes only the slave ends of the PTY pairs.
func (r *ptyStdioHandler) closeSlaves() error {
	if err := closeIfNotNil(r.stdinSlave); err != nil {
		return err
	}

	if err := closeIfNotNil(r.stdoutSlave); err != nil {
		return err
	}

	if err := closeIfNotNil(r.stderrSlave); err != nil {
		return err
	}

	return nil
}

// closeMasters closes only the master ends of the PTY pairs.
func (r *ptyStdioHandler) closeMasters() error {
	if err := closeIfNotNil(r.stdinMaster); err != nil {
		return err
	}

	if err := closeIfNotNil(r.stdoutMaster); err != nil {
		return err
	}

	if err := closeIfNotNil(r.stderrMaster); err != nil {
		return err
	}

	return nil
}
