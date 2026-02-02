package buffered_pipe

import (
	"io"
	"sync"
)

// BufferedPipe wraps io.Pipe with non-blocking writes
type BufferedPipe struct {
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	wg         sync.WaitGroup
	closeOnce  sync.Once
}

// NewBufferedPipe creates a new BufferedPipe
func NewBufferedPipe() *BufferedPipe {
	pr, pw := io.Pipe()
	return &BufferedPipe{
		pipeReader: pr,
		pipeWriter: pw,
	}
}

// Write queues data for writing in a goroutine. Never blocks the caller.
func (bp *BufferedPipe) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Copy data since caller may reuse the buffer
	data := make([]byte, len(p))
	copy(data, p)

	// Track this write
	bp.wg.Add(1)

	// Write in background goroutine
	go func() {
		defer bp.wg.Done()
		bp.pipeWriter.Write(data)
	}()

	return len(p), nil
}

// Read reads data from the pipe. Blocks until data is available.
// Returns io.EOF when pipe is closed and all data has been read.
func (bp *BufferedPipe) Read(p []byte) (n int, err error) {
	return bp.pipeReader.Read(p)
}

// CloseWrite waits for all pending writes to complete, then closes the pipe
func (bp *BufferedPipe) CloseWrite() error {
	bp.closeOnce.Do(func() {
		bp.wg.Wait()          // Wait for all Write() goroutines to finish
		bp.pipeWriter.Close() // Close the write side
	})
	return nil
}
