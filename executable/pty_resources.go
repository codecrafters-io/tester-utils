package executable

import (
	"os"

	"github.com/creack/pty"
)

// ptyResources manages the three PTY pairs used for stdin, stdout, and stderr
type ptyResources struct {
	stdinMaster, stdinSlave   *os.File
	stdoutMaster, stdoutSlave *os.File
	stderrMaster, stderrSlave *os.File
}

// openAll attempts to open all three PTY pairs.
// Returns an error if any PTY fails to open, and automatically cleans up any successfully opened PTYs.
func (r *ptyResources) openAll() error {
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
func (r *ptyResources) closeAll() {
	if r.stdinMaster != nil {
		r.stdinMaster.Close()
	}
	if r.stdinSlave != nil {
		r.stdinSlave.Close()
	}
	if r.stdoutMaster != nil {
		r.stdoutMaster.Close()
	}
	if r.stdoutSlave != nil {
		r.stdoutSlave.Close()
	}
	if r.stderrMaster != nil {
		r.stderrMaster.Close()
	}
	if r.stderrSlave != nil {
		r.stderrSlave.Close()
	}
}

// closeSlaves closes only the slave ends of the PTY pairs.
// This is called after cmd.Start() succeeds to remove references in the parent process.
func (r *ptyResources) closeSlaves() {
	if r.stdinSlave != nil {
		r.stdinSlave.Close()
	}
	if r.stdoutSlave != nil {
		r.stdoutSlave.Close()
	}
	if r.stderrSlave != nil {
		r.stderrSlave.Close()
	}
}
