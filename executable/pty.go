package executable

/*
#include <stdlib.h>
#ifdef __APPLE__
#include <util.h>
#elif __linux__
#include <pty.h>
#endif

int open_pty(int *master_fd, int *slave_fd) {
    return openpty(master_fd, slave_fd, NULL, NULL, NULL);
}
*/
import "C"
import (
	"fmt"
	"os"
)

// openpty creates a new pseudo-terminal pair using platform C openpty()
func openpty() (master, slave *os.File, err error) {
	var masterFd, slaveFd C.int

	if C.open_pty(&masterFd, &slaveFd) == -1 {
		return nil, nil, fmt.Errorf("openpty failed")
	}

	return os.NewFile(uintptr(masterFd), "master"),
		os.NewFile(uintptr(slaveFd), "slave"),
		nil
}
