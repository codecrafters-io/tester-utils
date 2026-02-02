package stdio_handler

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

type SinglePtyStdioHandler struct {
	Width  uint
	Height uint
	master *os.File
	slave  *os.File
}

func (h *SinglePtyStdioHandler) SetupStreams(cmd *exec.Cmd) error {
	if h.Width == 0 || h.Height == 0 {
		panic("Codecrafters Internal Error - SinglePtyStdioHandler:SetupStreams: Height and Width of PTY not initialized")
	}

	var err error
	h.master, h.slave, err = pty.Open()

	if err != nil {
		return err
	}

	pty.Setsize(h.master, &pty.Winsize{
		Rows: uint16(h.Height),
		Cols: uint16(h.Width),
	})

	pty.Setsize(h.slave, &pty.Winsize{
		Rows: uint16(h.Height),
		Cols: uint16(h.Width),
	})

	cmd.Stdin = h.slave
	cmd.Stdout = h.slave
	cmd.Stderr = h.slave

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Use the configuration used by creack/pty: https://github.com/creack/pty/blob/master/start.go#L19
	cmd.SysProcAttr.Setsid = true
	cmd.SysProcAttr.Setctty = true
	cmd.SysProcAttr.Setpgid = false // process group ID cannot be changed for session leader

	return nil
}

func (h *SinglePtyStdioHandler) GetStdin() io.WriteCloser {
	return h.master
}

func (h *SinglePtyStdioHandler) GetStdout() io.ReadCloser {
	return h.master
}

// GetStderr deliberately returns stderr as a simple no-op closer
// This is because in a single PTY setup the same device is used as stdout and stderr
// Returning the same device will result in a race condition if the caller
// uses both stdout and stderr to read separately
func (h *SinglePtyStdioHandler) GetStderr() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(nil))
}

func (h *SinglePtyStdioHandler) CloseChildStreams() error {
	return h.slave.Close()
}

func (h *SinglePtyStdioHandler) CloseParentStreams() error {
	return h.master.Close()
}

func (h *SinglePtyStdioHandler) TerminateStdin() error {
	_, err := h.master.Write([]byte("\n\004"))
	return err
}

func (h *SinglePtyStdioHandler) Clone() StdioHandler {
	return &SinglePtyStdioHandler{
		Width:  h.Width,
		Height: h.Height,
	}
}
