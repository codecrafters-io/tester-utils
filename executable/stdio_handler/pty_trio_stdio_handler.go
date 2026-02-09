package stdio_handler

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// PtyTrioStdioHandler deals with PTY based i/o
type PtyTrioStdioHandler struct {
	stdoutMaster, stdoutSlave *os.File
	stderrMaster, stderrSlave *os.File
	stdinMaster, stdinSlave   *os.File
	DisableAutomaticIORelay   bool
}

func (h *PtyTrioStdioHandler) GetStdin() io.WriteCloser {
	return h.stdinMaster
}

func (h *PtyTrioStdioHandler) GetStdout() io.ReadCloser {
	return h.stdoutMaster
}

func (h *PtyTrioStdioHandler) GetStderr() io.ReadCloser {
	return h.stderrMaster
}

func (h *PtyTrioStdioHandler) SetupStreams(cmd *exec.Cmd) error {
	if err := h.openAll(); err != nil {
		return err
	}

	// Assign slave end of PTYs to the child process
	cmd.Stdin = h.stdinSlave
	cmd.Stdout = h.stdoutSlave
	cmd.Stderr = h.stderrSlave

	return nil
}

func (h *PtyTrioStdioHandler) CloseChildStreams() error {
	// Close slave ends - child process now owns them
	return h.closeSlaves()
}

func (h *PtyTrioStdioHandler) CloseParentStreams() error {
	return h.closeMasters()
}

func (h *PtyTrioStdioHandler) TerminateStdin() error {
	// Send (\n + Ctrl-D) for closing input stream
	_, err := h.stdinMaster.Write([]byte("\n\004"))
	return err
}

func (h *PtyTrioStdioHandler) Clone() StdioHandler {
	return &PtyTrioStdioHandler{
		DisableAutomaticIORelay: h.DisableAutomaticIORelay,
	}
}

func (h *PtyTrioStdioHandler) NeedsIORelaySetup() bool {
	return !h.DisableAutomaticIORelay
}

// openAll attempts to open all three PTY pairs.
// Returns an error if any PTY fails to open, and automatically cleans up any successfully opened PTYs.
func (h *PtyTrioStdioHandler) openAll() error {
	var err error

	h.stdinMaster, h.stdinSlave, err = pty.Open()
	if err != nil {
		return err
	}

	h.stdoutMaster, h.stdoutSlave, err = pty.Open()
	if err != nil {
		h.closeAll()
		return err
	}

	h.stderrMaster, h.stderrSlave, err = pty.Open()
	if err != nil {
		h.closeAll()
		return err
	}

	return nil
}

// closeAll closes all PTY file descriptors.
func (h *PtyTrioStdioHandler) closeAll() error {
	var firstError error

	// best effort
	if closeMasterError := h.closeMasters(); closeMasterError != nil {
		firstError = closeMasterError
	}

	if closeSlaveError := h.closeSlaves(); closeSlaveError != nil && firstError == nil {
		firstError = closeSlaveError
	}

	return firstError
}

// closeSlaves closes only the slave ends of the PTY pairs.
func (h *PtyTrioStdioHandler) closeSlaves() error {
	// PTY are managed by ptyStdioHandler alone, and are not modified externally, so
	// closeIfOpen() is not needed here
	return closeAllWithCloserFunc(closeIfNotNil, h.stdinSlave, h.stdoutSlave, h.stderrSlave)
}

// closeMasters closes only the master ends of the PTY pairs.
func (h *PtyTrioStdioHandler) closeMasters() error {
	return closeAllWithCloserFunc(closeIfNotNil, h.stdinMaster, h.stdoutMaster, h.stderrMaster)
}
