package buffered_pipe

import (
	"io"
	"sync"
)

// BufferedPipe is a non-blocking pipe with ordered writes
type BufferedPipe struct {
	buffer     chan []byte
	pipeClosed chan struct{}
	once       sync.Once
	pending    []byte // stores remaining bytes from partial reads
}

// NewBufferedPipe creates a new BufferedPipe with the specified buffer size
func NewBufferedPipe(bufferSize int) *BufferedPipe {
	return &BufferedPipe{
		buffer:     make(chan []byte, bufferSize),
		pipeClosed: make(chan struct{}),
	}
}

// Write queues data for writing. Never blocks.
// Returns error only if pipe is closed.
func (bp *BufferedPipe) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Copy data since caller may reuse the buffer
	data := make([]byte, len(p))
	copy(data, p)

	select {
	case bp.buffer <- data:
		return len(p), nil
	case <-bp.pipeClosed:
		return 0, io.ErrClosedPipe
	default:
		// Buffer full - drop data but return success
		return len(p), nil
	}
}

// Read reads data from the pipe. Blocks until data is available.
// Returns io.EOF when pipe is closed and no data remains.
func (bp *BufferedPipe) Read(p []byte) (n int, err error) {
	// First, return any pending bytes from a previous partial read
	if len(bp.pending) > 0 {
		n = copy(p, bp.pending)
		bp.pending = bp.pending[n:]
		return n, nil
	}

	select {
	case data, ok := <-bp.buffer:
		if !ok {
			return 0, io.EOF
		}
		n = copy(p, data)
		if n < len(data) {
			bp.pending = data[n:]
		}
		return n, nil
	case <-bp.pipeClosed:
		// Pipe closed, check if any data remains
		select {
		case data, ok := <-bp.buffer:
			if !ok {
				return 0, io.EOF
			}
			n = copy(p, data)
			if n < len(data) {
				bp.pending = data[n:]
			}
			return n, nil
		default:
			return 0, io.EOF
		}
	}
}

// CloseWrite closes the write side of the pipe
func (bp *BufferedPipe) CloseWrite() error {
	bp.once.Do(func() {
		close(bp.pipeClosed)
		close(bp.buffer)
	})
	return nil
}
