//go:build linux

package executable

/*
#include <stdlib.h>
#include <pty.h>

int open_pty(int *master_fd, int *slave_fd) {
    return openpty(master_fd, slave_fd, NULL, NULL, NULL);
}
*/
import "C"
