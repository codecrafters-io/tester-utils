package buffered_pipe

import (
	"io"
	"sync"
)

// BufferedPipe is a pipe that preserves write order while handling slow/missing readers.
// Writes are queued and processed sequentially, preventing blocking of the writer.
type BufferedPipe struct {
	writeBuffer chan []byte
	readBuffer  chan []byte
	done        chan struct{}
	closeOnce   sync.Once
	wg          sync.WaitGroup

	// For partial reads
	partial       []byte
	partialOffset int
	mu            sync.Mutex
	readMu        sync.Mutex // serializes Read calls to prevent race conditions with concurrent readers
}

// NewBufferedPipe creates a new BufferedPipe with the specified buffer size.
func NewBufferedPipe(bufferSize int) *BufferedPipe {
	bp := &BufferedPipe{
		writeBuffer: make(chan []byte, bufferSize),
		readBuffer:  make(chan []byte, bufferSize),
		done:        make(chan struct{}),
	}

	// Start the sequential transfer goroutine
	bp.wg.Add(1)
	go bp.transferLoop()

	return bp
}

// transferLoop processes writes sequentially, preserving order
func (bp *BufferedPipe) transferLoop() {
	defer bp.wg.Done()
	defer close(bp.readBuffer)

	for {
		select {
		case data, ok := <-bp.writeBuffer:
			if !ok {
				// Write channel closed, we're done
				return
			}
			// Transfer data to read buffer
			// This blocks if reader is slow, but that's fine
			// because we're in a dedicated goroutine
			select {
			case bp.readBuffer <- data:
				// Successfully transferred
			case <-bp.done:
				// Pipe closed, drain remaining
				for range bp.writeBuffer {
					// Discard remaining queued writes
				}
				return
			}
		case <-bp.done:
			return
		}
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
	case bp.writeBuffer <- data:
		// Successfully queued
		return len(p), nil
	case <-bp.done:
		return 0, io.ErrClosedPipe
	default:
		// Buffer full - drop data but report success
		// This prevents blocking the relay goroutine
		return len(p), nil
	}
}

// Read reads data from the pipe. Blocks until data is available or pipe is closed.
// Implements standard io.Reader semantics with proper partial read handling.
func (bp *BufferedPipe) Read(p []byte) (n int, err error) {
	bp.readMu.Lock()
	defer bp.readMu.Unlock()

	bp.mu.Lock()
	defer bp.mu.Unlock()

	// First, try to consume any leftover data from previous read
	if bp.partialOffset < len(bp.partial) {
		n = copy(p, bp.partial[bp.partialOffset:])
		bp.partialOffset += n

		// If we've consumed all of partial, clear it
		if bp.partialOffset >= len(bp.partial) {
			bp.partial = nil
			bp.partialOffset = 0
		}

		return n, nil
	}

	// No leftover data, get next chunk from channel (blocking)
	bp.mu.Unlock()
	chunk, ok := <-bp.readBuffer
	bp.mu.Lock()

	if !ok {
		// Channel closed and empty
		return 0, io.EOF
	}

	// Copy as much as fits into p
	n = copy(p, chunk)

	// If chunk is larger than p, save the remainder for next Read()
	if n < len(chunk) {
		bp.partial = chunk
		bp.partialOffset = n
	}

	return n, nil
}

// Close closes the write side of the pipe and waits for all queued writes to complete.
func (bp *BufferedPipe) Close() error {
	bp.closeOnce.Do(func() {
		close(bp.writeBuffer)
		bp.wg.Wait() // Wait for transferLoop to finish
		close(bp.done)
	})
	return nil
}
