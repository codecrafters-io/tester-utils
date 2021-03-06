package tester_utils

import (
	"bytes"
	"errors"
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

	// These are set & removed together
	cmd              *exec.Cmd
	stdoutPipe       io.ReadCloser
	stderrPipe       io.ReadCloser
	stdoutBytes      []byte
	stderrBytes      []byte
	stdoutBuffer     *bytes.Buffer
	stderrBuffer     *bytes.Buffer
	stdoutLineWriter *linewriter.LineWriter
	stderrLineWriter *linewriter.LineWriter
	readDone         chan bool
}

// ExecutableResult holds the result of an executable run
type ExecutableResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type loggerWriter struct {
	loggerFunc func(string)
	buffer     []byte
}

func newLoggerWriter(loggerFunc func(string)) *loggerWriter {
	return &loggerWriter{
		loggerFunc: loggerFunc,
		buffer:     make([]byte, 80),
	}
}

func (w *loggerWriter) Write(bytes []byte) (n int, err error) {
	w.loggerFunc(string(bytes[:len(bytes)-1]))
	return len(bytes), nil
}

func nullLogger(msg string) {
	return
}

// newExecutable returns an Executable
func newExecutable(path string) *Executable {
	return &Executable{path: path, timeoutInSecs: 10, loggerFunc: nullLogger}
}

// newVerboseExecutable returns an Executable struct with a logger configured
func newVerboseExecutable(path string, loggerFunc func(string)) *Executable {
	return &Executable{path: path, timeoutInSecs: 10, loggerFunc: loggerFunc}
}

func (e *Executable) isRunning() bool {
	return e.cmd != nil
}

// Start starts the specified command but does not wait for it to complete.
func (e *Executable) Start(args ...string) error {
	var err error

	if e.isRunning() {
		return errors.New("process already in progress")
	}

	// TODO: Use timeout!
	e.cmd = exec.Command(e.path, args...)
	e.cmd.Dir = e.WorkingDir
	e.readDone = make(chan bool)

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

	err = e.cmd.Start()
	if err != nil {
		return err
	}

	e.setupIORelay(e.stdoutPipe, e.stdoutBuffer, e.stdoutLineWriter)
	e.setupIORelay(e.stderrPipe, e.stderrBuffer, e.stderrLineWriter)

	return nil
}

func (e *Executable) setupIORelay(childReader io.Reader, buffer io.Writer, writer io.Writer) {
	go func() {
		writer := io.MultiWriter(writer, buffer)
		_, err := io.Copy(writer, childReader)
		if err != nil {
			panic(err)
		}
		e.readDone <- true
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

// Wait waits for the program to finish and results the result
func (e *Executable) Wait() (ExecutableResult, error) {
	defer func() {
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
	}()

	<-e.readDone
	<-e.readDone

	err := e.cmd.Wait()
	e.stdoutLineWriter.Flush()
	e.stderrLineWriter.Flush()

	if err != nil {
		// Ignore exit errors, we'd rather send the exit code back
		if _, ok := err.(*exec.ExitError); !ok {
			return ExecutableResult{}, err
		}
	}

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
	syscall.Kill(e.cmd.Process.Pid, syscall.SIGTERM)
	_, err := e.Wait()
	return err
}
