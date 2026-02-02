package executable

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"io"
	"os/exec"
	"syscall"

	"github.com/codecrafters-io/tester-utils/executable/buffered_pipe"
	"github.com/codecrafters-io/tester-utils/executable/stdio_handler"
	"github.com/codecrafters-io/tester-utils/linewriter"
	"go.chromium.org/luci/common/system/environ"
)

// Executable represents a program that can be executed
type Executable struct {
	// Path is the path to the executable.
	Path string

	// TimeoutInMilliseconds is the maximum time the process can run.
	TimeoutInMilliseconds int

	// MemoryLimitInBytes sets the maximum memory the process can use (Linux only).
	// If exceeded, the process will be killed and an error will be returned.
	// Defaults to 2GB. Set to 0 to disable memory limiting.
	MemoryLimitInBytes int64

	// StdioHandler dictates how an executable's stdin/stdout/stderr are set up
	// For more info, please see the stdio_handler module
	StdioHandler stdio_handler.StdioHandler

	// WorkingDir can be set before calling Start or Run to customize the working directory of the executable.
	WorkingDir string

	// Env contains environment variables required for the executable
	Env environ.Env

	// Process is the os.Process object for the executable.
	// TODO: See if this actually needs to be exported?
	Process *os.Process

	// loggerFunc is the function called w/ output from the executable.
	loggerFunc func(string)

	// These are set & removed together
	atleastOneReadDone bool
	memoryMonitor      *memoryMonitor // Monitors process memory usage and kills if limit exceeded
	cmd                *exec.Cmd
	ctxCancelFunc      context.CancelFunc
	ctxWithTimeout     context.Context
	readDone           chan bool

	stderrBuffer     *bytes.Buffer
	stderrBytes      []byte
	stderrLineWriter *linewriter.LineWriter
	stderrStream     *buffered_pipe.BufferedPipe

	stdoutBuffer     *bytes.Buffer
	stdoutBytes      []byte
	stdoutLineWriter *linewriter.LineWriter
	stdoutStream     *buffered_pipe.BufferedPipe
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
	return &Executable{
		Path:                  e.Path,
		TimeoutInMilliseconds: e.TimeoutInMilliseconds,
		loggerFunc:            e.loggerFunc,
		WorkingDir:            e.WorkingDir,
		StdioHandler:          e.StdioHandler.Clone(),
		MemoryLimitInBytes:    e.MemoryLimitInBytes,
	}
}

// NewExecutable returns an Executable
func NewExecutable(path string) *Executable {
	return &Executable{
		Path:                  path,
		TimeoutInMilliseconds: 10 * 1000,
		loggerFunc:            nullLogger,
		StdioHandler:          &stdio_handler.PipeTrioStdioHandler{},
		MemoryLimitInBytes:    GetMemoryLimitInBytes(),
	}
}

// NewVerboseExecutable returns an Executable struct with a logger configured
func NewVerboseExecutable(path string, loggerFunc func(string)) *Executable {
	return &Executable{
		Path:                  path,
		TimeoutInMilliseconds: 10 * 1000,
		loggerFunc:            loggerFunc,
		StdioHandler:          &stdio_handler.PipeTrioStdioHandler{},
		MemoryLimitInBytes:    GetMemoryLimitInBytes(),
	}
}

func (e *Executable) isRunning() bool {
	return e.cmd != nil
}

func (e *Executable) HasExited() bool {
	return e.atleastOneReadDone
}

func (e *Executable) GetStdoutStream() *buffered_pipe.BufferedPipe {
	return e.stdoutStream
}

// Start starts the specified command but does not wait for it to complete.
func (e *Executable) Start(args ...string) error {
	var err error

	if e.isRunning() {
		return errors.New("process already in progress")
	}

	// Get the absolute path for e.Path
	absolutePath, err := resolveAbsolutePath(e.Path)

	if err != nil {
		return fmt.Errorf("%s not found", filepath.Base(e.Path))
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

	var commandName string

	// If e.Path is a relative path -> use the absolute path for launching the command, else use e.Path
	if strings.Contains(e.Path, "/") {
		commandName = absolutePath
	} else {
		commandName = e.Path
	}

	cmd := exec.CommandContext(ctx, commandName, args...)
	cmd.Env = e.initializeSafeEnvVars()
	cmd.Dir = e.WorkingDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	e.memoryMonitor = newMemoryMonitor(e.MemoryLimitInBytes)

	e.readDone = make(chan bool)
	e.atleastOneReadDone = false

	e.stdoutBytes = []byte{}
	e.stdoutBuffer = bytes.NewBuffer(e.stdoutBytes)
	e.stdoutLineWriter = linewriter.New(newLoggerWriter(e.loggerFunc), 500*time.Millisecond)
	e.stdoutStream = buffered_pipe.NewBufferedPipe(30000)

	e.stderrBytes = []byte{}
	e.stderrBuffer = bytes.NewBuffer(e.stderrBytes)
	e.stderrLineWriter = linewriter.New(newLoggerWriter(e.loggerFunc), 500*time.Millisecond)
	e.stderrStream = buffered_pipe.NewBufferedPipe(30000)

	// Setup standard streams
	if err := e.StdioHandler.SetupStreams(cmd); err != nil {
		return err
	}

	err = cmd.Start()
	// Close child streams after cmd.Start() regardless of success/failure
	// cmd.Start() duplicates streams to child, we can close our duplicated copies
	e.StdioHandler.CloseChildStreams()

	// In case of error, close parent's streams as well
	defer func() {
		if err != nil {
			e.StdioHandler.CloseParentStreams()
		}
	}()

	if err != nil {
		return err
	}

	e.Process, err = os.FindProcess(cmd.Process.Pid)
	if err != nil {
		return err
	}

	// At this point, it is safe to set e.cmd as cmd, if any of the above steps fail, we don't want to leave e.cmd in an inconsistent state
	e.cmd = cmd

	// Start memory monitoring for RSS-based memory limiting (Linux only, no-op on other platforms)
	e.memoryMonitor.start(cmd.Process.Pid)

	e.setupIORelay(e.StdioHandler.GetStdout(), e.stdoutBuffer, e.stdoutStream, e.stdoutLineWriter)
	e.setupIORelay(e.StdioHandler.GetStderr(), e.stderrBuffer, e.stderrStream, e.stderrLineWriter)

	return nil
}

func (e *Executable) setupIORelay(source io.Reader, buffer *bytes.Buffer, stream *buffered_pipe.BufferedPipe, lineWriter *linewriter.LineWriter) {
	go func() {
		defer stream.Close()

		combinedDestination := io.MultiWriter(buffer, lineWriter, stream)
		bytesWritten, err := io.Copy(combinedDestination, io.LimitReader(source, 30000))

		if err != nil {
			if !(isTTY(source) && errors.Is(err, syscall.EIO)) {
				panic(err)
			}
		}

		if bytesWritten == 30000 {
			e.loggerFunc("Warning: Logs exceeded allowed limit, output might be truncated.\n")
		}

		e.atleastOneReadDone = true
		e.readDone <- true
		io.Copy(io.Discard, source)
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

	e.StdioHandler.GetStdin().Write(stdin)

	return e.Wait()
}

// formatBytesHumanReadable formats bytes as a human-readable string (e.g., "50 MB", "2 GB")
func formatBytesHumanReadable(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%d GB", bytes/GB)
	case bytes >= MB:
		return fmt.Sprintf("%d MB", bytes/MB)
	case bytes >= KB:
		return fmt.Sprintf("%d KB", bytes/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// ErrMemoryLimitExceeded is returned when a process exceeds its memory limit
var ErrMemoryLimitExceeded = errors.New("process exceeded memory limit")

// Wait waits for the program to finish and returns the result.
func (e *Executable) Wait() (ExecutableResult, error) {
	defer func() {
		e.ctxCancelFunc()

		e.memoryMonitor.stop()
		e.StdioHandler.CloseParentStreams()

		e.atleastOneReadDone = false
		e.cmd = nil
		e.ctxCancelFunc = nil
		e.ctxWithTimeout = nil
		e.memoryMonitor = nil

		e.stdoutBuffer = nil
		e.stdoutBytes = nil
		e.stdoutLineWriter = nil
		e.stdoutStream = nil

		e.stderrBuffer = nil
		e.stderrBytes = nil
		e.stderrLineWriter = nil
		e.stderrStream = nil

		e.readDone = nil
		e.StdioHandler = e.StdioHandler.Clone()
	}()

	e.StdioHandler.TerminateStdin()

	<-e.readDone
	<-e.readDone

	e.stdoutStream.Close()
	e.stderrStream.Close()

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

	// Check if process was killed due to OOM (exit code 137 = 128 + SIGKILL)
	if e.memoryMonitor.wasOOMKilled() {
		return result, fmt.Errorf("process exceeded memory limit (%s): %w", formatBytesHumanReadable(e.MemoryLimitInBytes), ErrMemoryLimitExceeded)
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

// initializeSafeEnvVars initializes environment variables for the executable
// Environment variables starting with CODECRAFTERS_SECRET are filtered out
func (e *Executable) initializeSafeEnvVars() []string {
	if e.Env.Len() == 0 {
		e.Env = environ.New(os.Environ())
	}

	allEnvVars := e.Env.Sorted()
	safeEnvVars := make([]string, 0, len(allEnvVars))

	for _, envVar := range allEnvVars {
		// Filter out environment variables starting with `CODECRAFTERS_SECRET`
		if !strings.HasPrefix(envVar, "CODECRAFTERS_SECRET") {
			safeEnvVars = append(safeEnvVars, envVar)
		}
	}

	return safeEnvVars
}
