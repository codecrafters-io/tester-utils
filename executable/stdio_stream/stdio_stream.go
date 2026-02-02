package stdio_stream

import (
	"net"
	"os"
	"sync"
	"syscall"
)

// StdioStream uses Unix domain sockets for buffered, non-blocking writes
type StdioStream struct {
	writeConn  *net.UnixConn
	readConn   *net.UnixConn
	writeQueue chan []byte
	wg         sync.WaitGroup
	closeOnce  sync.Once
}

// NewStdioStream creates a new BufferedPipe with specified buffer size
func NewStdioStream(bufferSize int) *StdioStream {
	// Create socketpair (like pipe but with kernel buffering)
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}

	// Set buffer sizes on the socket
	syscall.SetsockoptInt(fds[0], syscall.SOL_SOCKET, syscall.SO_SNDBUF, 65536)
	syscall.SetsockoptInt(fds[1], syscall.SOL_SOCKET, syscall.SO_RCVBUF, 65536)

	// Convert to net.Conn
	writeFile := os.NewFile(uintptr(fds[0]), "write")
	readFile := os.NewFile(uintptr(fds[1]), "read")

	writeConn, _ := net.FileConn(writeFile)
	readConn, _ := net.FileConn(readFile)

	writeFile.Close()
	readFile.Close()

	bp := &StdioStream{
		writeConn:  writeConn.(*net.UnixConn),
		readConn:   readConn.(*net.UnixConn),
		writeQueue: make(chan []byte, bufferSize),
	}

	// Start writer loop
	bp.wg.Add(1)
	go bp.writerLoop()

	return bp
}

// writerLoop processes writes sequentially
func (bp *StdioStream) writerLoop() {
	defer bp.wg.Done()
	defer bp.writeConn.Close()

	for data := range bp.writeQueue {
		bp.writeConn.Write(data) // Won't block due to kernel buffering
	}
}

// Write queues data for writing. Never blocks the caller.
// If buffer is full, drops the data but returns success.
func (bp *StdioStream) Write(p []byte) (n int, err error) {
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
		// Buffer full - drop data but return success
		return len(p), nil
	}
}

// Read reads data from the pipe. Blocks until data is available.
func (bp *StdioStream) Read(p []byte) (n int, err error) {
	return bp.readConn.Read(p)
}

// CloseWrite waits for all queued writes to complete, then closes
func (bp *StdioStream) CloseWrite() error {
	bp.closeOnce.Do(func() {
		close(bp.writeQueue) // No more writes accepted
		bp.wg.Wait()         // Wait for writerLoop (won't hang due to socket buffering)
		bp.writeConn.Close()
		bp.readConn.Close()
	})
	return nil
}
