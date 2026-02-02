package stdio_handler

import (
	"io"
	"os/exec"
)

// PipeTrioStdioHandler deals with pipe based i/o
type PipeTrioStdioHandler struct {
	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
}

func (h *PipeTrioStdioHandler) GetStdin() io.WriteCloser {
	return h.stdinPipe
}

func (h *PipeTrioStdioHandler) GetStdout() io.ReadCloser {
	return h.stdoutPipe
}

func (h *PipeTrioStdioHandler) GetStderr() io.ReadCloser {
	return h.stderrPipe
}

func (h *PipeTrioStdioHandler) SetupStreams(cmd *exec.Cmd) error {
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

func (h *PipeTrioStdioHandler) CloseChildStreams() error {
	// No action needed here: closing child streams is automatically handled by exec library
	return nil
}

func (h *PipeTrioStdioHandler) CloseParentStreams() error {
	return closeAllWithCloserFunc(closeIfOpen, h.stdinPipe, h.stdoutPipe, h.stderrPipe)
}

func (h *PipeTrioStdioHandler) TerminateStdin() error {
	if err := h.stdinPipe.Close(); err != nil {
		return err
	}

	return nil
}

func (h *PipeTrioStdioHandler) Clone() StdioHandler {
	return &PipeTrioStdioHandler{}
}
