package buffered_pipe

import (
	"io"
	"sync"
)

// BufferedPipe wraps io.Pipe with non-blocking, ordered writes
type BufferedPipe struct {
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	writeQueue chan []byte
	wg         sync.WaitGroup
	closeOnce  sync.Once
}

// NewBufferedPipe creates a new BufferedPipe with specified buffer size
func NewBufferedPipe(bufferSize int) *BufferedPipe {
	pr, pw := io.Pipe()
	bp := &BufferedPipe{
		pipeReader: pr,
		pipeWriter: pw,
		writeQueue: make(chan []byte, bufferSize),
	}

	// Start single writer goroutine to preserve order
	bp.wg.Add(1)
	go bp.writerLoop()

	return bp
}

// writerLoop processes writes sequentially
func (bp *BufferedPipe) writerLoop() {
	defer bp.wg.Done()
	defer bp.pipeWriter.Close()

	for data := range bp.writeQueue {
		bp.pipeWriter.Write(data)
	}
}

// Write queues data for writing. Never blocks the caller.
// If buffer is full, drops the data but returns success.
func (bp *BufferedPipe) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Copy data since caller may reuse the buffer
	data := make([]byte, len(p))
	copy(data, p)

	select {
	case bp.writeQueue <- data:
		return len(p), nil
	default:
		return len(p), nil
	}
}

// Read reads data from the pipe. Blocks until data is available.
func (bp *BufferedPipe) Read(p []byte) (n int, err error) {
	return bp.pipeReader.Read(p)
}

// CloseWrite waits for all queued writes to complete, then closes the pipe
func (bp *BufferedPipe) CloseWrite() error {
	bp.closeOnce.Do(func() {
		close(bp.writeQueue) // No more writes accepted
		bp.wg.Wait()         // Wait for writerLoop to finish
	})
	return nil
}
