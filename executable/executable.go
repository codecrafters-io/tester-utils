package executable

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"io"
	"os/exec"
	"syscall"

	"github.com/codecrafters-io/tester-utils/linewriter"
)

// Executable represents a program that can be executed
type Executable struct {
	Path                  string
	TimeoutInMilliseconds int
	loggerFunc            func(string)

	ctxWithTimeout context.Context
	ctxCancelFunc  context.CancelFunc

	// WorkingDir can be set before calling Start or Run to customize the working directory of the executable.
	WorkingDir string

	Process *os.Process

	// These are set & removed together
	atleastOneReadDone bool
	cmd                *exec.Cmd
	stdoutBytes        []byte
	stderrBytes        []byte
	stdoutBuffer       *bytes.Buffer
	stderrBuffer       *bytes.Buffer
	stdoutLineWriter   *linewriter.LineWriter
	stderrLineWriter   *linewriter.LineWriter
	readDone           chan bool

	stdioHandler stdioHandler
}

// ExecutableResult holds the result of an executable run
type ExecutableResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type loggerWriter struct {
	loggerFunc func(string)
}

func newLoggerWriter(loggerFunc func(string)) *loggerWriter {
	return &loggerWriter{
		loggerFunc: loggerFunc,
	}
}

func (w *loggerWriter) Write(bytes []byte) (n int, err error) {
	w.loggerFunc(string(bytes[:len(bytes)-1]))
	return len(bytes), nil
}

func nullLogger(msg string) {
}

func (e *Executable) Clone() *Executable {
	var clonedStdioHandler stdioHandler

	// Instantiate the stdioHandler based on the type used in the source executable
	switch e.stdioHandler.(type) {
	case *pipeStdioHandler:
		clonedStdioHandler = &pipeStdioHandler{}
	case *ptyStdioHandler:
		clonedStdioHandler = &ptyStdioHandler{}
	default:
		panic("Codecrafters Internal Error - stdioHandler type is unknown")
	}

	return &Executable{
		Path:                  e.Path,
		TimeoutInMilliseconds: e.TimeoutInMilliseconds,
		loggerFunc:            e.loggerFunc,
		WorkingDir:            e.WorkingDir,
		stdioHandler:          clonedStdioHandler,
	}
}

// NewExecutable returns an Executable
func NewExecutable(path string) *Executable {
	return &Executable{
		Path:                  path,
		TimeoutInMilliseconds: 10 * 1000,
		loggerFunc:            nullLogger,
		stdioHandler:          &pipeStdioHandler{}, // default stdio handler
	}
}

// NewVerboseExecutable returns an Executable struct with a logger configured
func NewVerboseExecutable(path string, loggerFunc func(string), usePTY bool) *Executable {
	var stdioHandler stdioHandler
	if usePTY {
		stdioHandler = &ptyStdioHandler{}
	} else {
		stdioHandler = &pipeStdioHandler{}
	}
	return &Executable{
		Path:                  path,
		TimeoutInMilliseconds: 10 * 1000,
		loggerFunc:            loggerFunc,
		stdioHandler:          stdioHandler,
	}
}

// SetUsePty switches stdiohandler for executable between pip/pty based on usePty flag
func (e *Executable) SetUsePty(usePty bool) {
	if usePty {
		e.stdioHandler = &ptyStdioHandler{}
	} else {
		e.stdioHandler = &pipeStdioHandler{}
	}
}

func (e *Executable) isRunning() bool {
	return e.cmd != nil
}

func (e *Executable) HasExited() bool {
	return e.atleastOneReadDone
}

// Start starts the specified command but does not wait for it to complete.
func (e *Executable) Start(args ...string) error {
	var err error

	if e.isRunning() {
		return errors.New("process already in progress")
	}

	var absolutePath, resolvedPath string

	// While passing executables present on PATH, filepath.Abs is unable to resolve their absolute path.
	// In those cases we use the path returned by LookPath.
	resolvedPath, err = exec.LookPath(e.Path)
	if err == nil {
		absolutePath = resolvedPath
	} else {
		absolutePath, err = filepath.Abs(e.Path)
		if err != nil {
			return fmt.Errorf("%s not found", filepath.Base(e.Path))
		}
	}
	fileInfo, err := os.Stat(absolutePath)
	if err != nil {
		return fmt.Errorf("%s not found", filepath.Base(e.Path))
	}

	// Check executable permission
	if fileInfo.Mode().Perm()&0111 == 0 || fileInfo.IsDir() {
		return fmt.Errorf("%s (resolved to %s) is not an executable file", e.Path, absolutePath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.TimeoutInMilliseconds)*time.Millisecond)
	e.ctxWithTimeout = ctx
	e.ctxCancelFunc = cancel

	cmd := exec.CommandContext(ctx, e.Path, args...)
	cmd.Dir = e.WorkingDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	e.readDone = make(chan bool)
	e.atleastOneReadDone = false

	e.stdoutBytes = []byte{}
	e.stdoutBuffer = bytes.NewBuffer(e.stdoutBytes)
	e.stdoutLineWriter = linewriter.New(newLoggerWriter(e.loggerFunc), 500*time.Millisecond)

	e.stderrBytes = []byte{}
	e.stderrBuffer = bytes.NewBuffer(e.stderrBytes)
	e.stderrLineWriter = linewriter.New(newLoggerWriter(e.loggerFunc), 500*time.Millisecond)

	// Setup standard streams
	if err := e.stdioHandler.SetupStreams(cmd); err != nil {
		return err
	}

	err = cmd.Start()

	// This can be placed in e.Wait()
	// However, cmd.Start() closes duplicated streams as soon as the process starts
	// So, we repeat the pattern
	e.stdioHandler.CloseDuplicatedStreamsOfChild()

	if err != nil {
		e.stdioHandler.CleanupStreamsOnStartFailure()
		return err
	}

	e.Process, err = os.FindProcess(cmd.Process.Pid)
	if err != nil {
		return err
	}

	// At this point, it is safe to set e.cmd as cmd, if any of the above steps fail, we don't want to leave e.cmd in an inconsistent state
	e.cmd = cmd
	e.setupIORelay(e.stdioHandler.GetStdout(), e.stdoutBuffer, e.stdoutLineWriter)
	e.setupIORelay(e.stdioHandler.GetStderr(), e.stderrBuffer, e.stderrLineWriter)

	return nil
}

func (e *Executable) setupIORelay(source io.Reader, destination1 io.Writer, destination2 io.Writer) {
	go func() {
		combinedDestination := io.MultiWriter(destination1, destination2)
		// Limit to 30KB (~250 lines at 120 chars per line)
		bytesWritten, err := io.Copy(combinedDestination, io.LimitReader(source, 30000))
		if err != nil {
			// In linux, if the source is a terminal device, read(2) results in EIO when the child process has exited and closed its slave end
			// (Source: The Linux Programming Interface Appendix F - 64.1)
			// This can be safely ignored
			if !(isTTY(source) && errors.Is(err, syscall.EIO)) {
				panic(err)
			}
		}

		if bytesWritten == 30000 {
			e.loggerFunc("Warning: Logs exceeded allowed limit, output might be truncated.\n")
		}

		e.atleastOneReadDone = true
		e.readDone <- true
		io.Copy(io.Discard, source) // Let's drain the stream in case any content is leftover
	}()
}

func (e *Executable) Run(args ...string) (ExecutableResult, error) {
	var err error

	if err = e.Start(args...); err != nil {
		return ExecutableResult{}, err
	}

	return e.Wait()
}

// RunWithStdin starts the specified command, sends input, waits for it to complete and returns the
// result.
func (e *Executable) RunWithStdin(stdin []byte, args ...string) (ExecutableResult, error) {
	var err error

	if err = e.Start(args...); err != nil {
		return ExecutableResult{}, err
	}

	e.stdioHandler.WriteToStdin(stdin)

	return e.Wait()
}

// Wait waits for the program to finish and returns the result.
func (e *Executable) Wait() (ExecutableResult, error) {
	defer func() {
		e.ctxCancelFunc()
		// We finally close the FDs used by the parent
		e.stdioHandler.CleanupStreamsAfterWait()
		e.atleastOneReadDone = false
		e.cmd = nil
		e.ctxCancelFunc = nil
		e.ctxWithTimeout = nil
		e.stdoutBytes = nil
		e.stderrBytes = nil
		e.stdoutLineWriter = nil
		e.stderrLineWriter = nil
		e.readDone = nil
	}()

	e.stdioHandler.SendEofToStdin()

	<-e.readDone
	<-e.readDone

	err := e.cmd.Wait()

	exitCode := e.cmd.ProcessState.ExitCode()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitCode == -1 {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					// If the process was terminated by a signal, extract the signal number
					if status.Signaled() {
						exitCode = 128 + int(status.Signal())
					}
				}
			}
		} else {
			// Ignore other exit errors, we'd rather send the exit code back
			return ExecutableResult{}, err
		}
	}

	e.stdoutLineWriter.Flush()
	e.stderrLineWriter.Flush()

	stdout := e.stdoutBuffer.Bytes()
	stderr := e.stderrBuffer.Bytes()

	result := ExecutableResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}

	if e.ctxWithTimeout.Err() == context.DeadlineExceeded {
		return ExecutableResult{}, fmt.Errorf("execution timed out")
	}
	return result, nil
}

// Kill terminates the program
func (e *Executable) Kill() error {
	if !e.isRunning() {
		return nil
	}

	doneChannel := make(chan error, 1)

	go func() {
		syscall.Kill(e.cmd.Process.Pid, syscall.SIGTERM)  // Don't know if this is required
		syscall.Kill(-e.cmd.Process.Pid, syscall.SIGTERM) // Kill the whole process group
		_, err := e.Wait()
		doneChannel <- err
	}()

	var err error
	select {
	case doneError := <-doneChannel:
		err = doneError
	case <-time.After(2 * time.Second):
		cmd := e.cmd
		if cmd != nil {
			err = fmt.Errorf("program failed to exit in 2 seconds after receiving sigterm")
			syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)  // Don't know if this is required
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) // Kill the whole process group

			<-doneChannel // Wait for Wait() to return
		}
	}

	return err
}

// setupStandardStreams connects process' stdin, stdout, and stderr with pipe/pty depending on whether or not e.ShouldTTY is set
// onSuccessCleanup should be run if the executable successfully starts
// onFailure should be run if the executable fails to start properly
// err is returned if an error is encountered while setting up the streams
// func (e *Executable) setupStandardStreams(cmd *exec.Cmd) (onCmdStartSuccessCleanup func(), onCmdStartFailureCleanup func(), err error) {
// 	var ptyResources ptyResources

// 	if e.ShouldUsePTY {
// 		if err = ptyResources.openAll(); err != nil {
// 			return nil, nil, err
// 		}

// 		// Cleanup FDs if Start does not succeed
// 		defer func() {
// 			if err != nil {
// 				ptyResources.closeAll()
// 			}
// 		}()

// 		// Assign master and slave ends to parent and child respectively
// 		cmd.Stdout = ptyResources.stdoutSlave
// 		cmd.Stderr = ptyResources.stderrSlave
// 		cmd.Stdin = ptyResources.stdinSlave
// 		e.stdoutStream = ptyResources.stdoutMaster
// 		e.stderrStream = ptyResources.stderrMaster
// 		e.stdinStream = ptyResources.stdinMaster

// 	} else {
// 		// Setup standard streams using the provided function
// 		e.stdoutStream, err = cmd.StdoutPipe()
// 		if err != nil {
// 			return nil, nil, err
// 		}

// 		e.stderrStream, err = cmd.StderrPipe()
// 		if err != nil {
// 			return nil, nil, err
// 		}

// 		e.stdinStream, err = cmd.StdinPipe()
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 	}

// 	// Close slave ends of pipes if Start() succeeds
// 	onCmdStartSuccessCleanup = func() {
// 		if e.ShouldUsePTY {
// 			ptyResources.closeSlaves()
// 		}
// 	}

// 	// Close all master and slaves if Start() fails
// 	onCmdStartFailureCleanup = func() {
// 		if e.ShouldUsePTY {
// 			ptyResources.closeAll()
// 		} else {
// 		}
// 	}

// 	return onCmdStartSuccessCleanup, onCmdStartFailureCleanup, nil
// }
