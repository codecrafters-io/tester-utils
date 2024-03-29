package executable

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
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
	path          string
	timeoutInSecs int
	loggerFunc    func(string)

	// WorkingDir can be set before calling Start or Run to customize the working directory of the executable.
	WorkingDir string

	StdinPipe io.WriteCloser

	// These are set & removed together
	atleastOneReadDone bool
	cmd                *exec.Cmd
	stdoutPipe         io.ReadCloser
	stderrPipe         io.ReadCloser
	stdoutBytes        []byte
	stderrBytes        []byte
	stdoutBuffer       *bytes.Buffer
	stderrBuffer       *bytes.Buffer
	stdoutLineWriter   *linewriter.LineWriter
	stderrLineWriter   *linewriter.LineWriter
	readDone           chan bool
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
	return
}

func (e *Executable) Clone() *Executable {
	return &Executable{
		path:          e.path,
		timeoutInSecs: e.timeoutInSecs,
		loggerFunc:    e.loggerFunc,
		WorkingDir:    e.WorkingDir,
	}
}

// NewExecutable returns an Executable
func NewExecutable(path string) *Executable {
	return &Executable{path: path, timeoutInSecs: 10, loggerFunc: nullLogger}
}

// NewVerboseExecutable returns an Executable struct with a logger configured
func NewVerboseExecutable(path string, loggerFunc func(string)) *Executable {
	return &Executable{path: path, timeoutInSecs: 10, loggerFunc: loggerFunc}
}

func (e *Executable) isRunning() bool {
	return e.cmd != nil
}

func (e *Executable) HasExited() bool {
	return e.atleastOneReadDone == true
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
	resolvedPath, err = exec.LookPath(e.path)
	if err == nil {
		absolutePath = resolvedPath
	} else {
		absolutePath, err = filepath.Abs(e.path)
		if err != nil {
			return fmt.Errorf("%s not found", e.path)
		}
	}
	fileInfo, err := os.Stat(absolutePath)
	if err != nil {
		return fmt.Errorf("%s not found", e.path)
	}

	// Check executable permission
	if fileInfo.Mode().Perm()&0111 == 0 || fileInfo.IsDir() {
		return fmt.Errorf("%s is not an executable file", e.path)
	}

	// TODO: Use timeout!
	e.cmd = exec.Command(e.path, args...)
	e.cmd.Dir = e.WorkingDir
	e.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	e.readDone = make(chan bool)
	e.atleastOneReadDone = false

	// Setup stdout capture
	e.stdoutPipe, err = e.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	e.stdoutBytes = []byte{}
	e.stdoutBuffer = bytes.NewBuffer(e.stdoutBytes)
	e.stdoutLineWriter = linewriter.New(newLoggerWriter(e.loggerFunc), 500*time.Millisecond)

	// Setup stderr relay
	e.stderrPipe, err = e.cmd.StderrPipe()
	if err != nil {
		return err
	}
	e.stderrBytes = []byte{}
	e.stderrBuffer = bytes.NewBuffer(e.stderrBytes)
	e.stderrLineWriter = linewriter.New(newLoggerWriter(e.loggerFunc), 500*time.Millisecond)

	e.StdinPipe, err = e.cmd.StdinPipe()
	if err != nil {
		return err
	}

	err = e.cmd.Start()
	if err != nil {
		return err
	}

	e.setupIORelay(e.stdoutPipe, e.stdoutBuffer, e.stdoutLineWriter)
	e.setupIORelay(e.stderrPipe, e.stderrBuffer, e.stderrLineWriter)

	return nil
}

func (e *Executable) setupIORelay(source io.Reader, destination1 io.Writer, destination2 io.Writer) {
	go func() {
		combinedDestination := io.MultiWriter(destination1, destination2)
		bytesWritten, err := io.Copy(combinedDestination, io.LimitReader(source, 1024*1024)) // 1MB
		if err != nil {
			panic(err)
		}

		if bytesWritten == 1024*1024 {
			e.loggerFunc("Warning: Logs exceeded allowed limit, output might be truncated.\n")
		}

		e.atleastOneReadDone = true
		e.readDone <- true
		io.Copy(ioutil.Discard, source) // Let's drain the pipe in case any content is leftover
	}()
}

// Run starts the specified command, waits for it to complete and returns the
// result.
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

	e.StdinPipe.Write(stdin)

	return e.Wait()
}

// Wait waits for the program to finish and results the result
func (e *Executable) Wait() (ExecutableResult, error) {
	defer func() {
		e.atleastOneReadDone = false
		e.cmd = nil
		e.stdoutPipe = nil
		e.stderrPipe = nil
		e.stdoutBuffer = nil
		e.stderrBuffer = nil
		e.stdoutBytes = nil
		e.stderrBytes = nil
		e.stdoutLineWriter = nil
		e.stderrLineWriter = nil
		e.readDone = nil
		e.StdinPipe = nil
	}()

	e.StdinPipe.Close()

	<-e.readDone
	<-e.readDone

	err := e.cmd.Wait()

	if err != nil {
		// Ignore exit errors, we'd rather send the exit code back
		if _, ok := err.(*exec.ExitError); !ok {
			return ExecutableResult{}, err
		}
	}

	e.stdoutLineWriter.Flush()
	e.stderrLineWriter.Flush()

	stdout := e.stdoutBuffer.Bytes()
	stderr := e.stderrBuffer.Bytes()

	return ExecutableResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: e.cmd.ProcessState.ExitCode(),
	}, nil
}

// Kill terminates the program
func (e *Executable) Kill() error {
	if !e.isRunning() {
		return nil
	}

	doneChannel := make(chan error, 1)

	go func() {
		processExists := e.cmd.Process.Signal(syscall.Signal(0))
		var err error
		if processExists != nil {
			syscall.Kill(e.cmd.Process.Pid, syscall.SIGTERM)  // Don't know if this is required
			syscall.Kill(-e.cmd.Process.Pid, syscall.SIGTERM) // Kill the whole process group
			_, err = e.Wait()
		}
		doneChannel <- err
	}()

	var err error
	select {
	case doneError := <-doneChannel:
		err = doneError
	case <-time.After(2 * time.Second):
		err = fmt.Errorf("program failed to exit in 2 seconds after receiving sigterm")
		// _, err := os.FindProcess(e.cmd.Process.Pid)
		processExists := e.cmd.Process.Signal(syscall.Signal(0))
		if processExists == nil {
			syscall.Kill(e.cmd.Process.Pid, syscall.SIGKILL)  // Don't know if this is required
			syscall.Kill(-e.cmd.Process.Pid, syscall.SIGKILL) // Kill the whole process group
		}

		<-doneChannel // Wait for Wait() to return
	}

	return err
}
