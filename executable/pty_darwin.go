//go:build darwin

package executable

/*
#include <stdlib.h>
#include <util.h>

int open_pty(int *master_fd, int *slave_fd) {
    return openpty(master_fd, slave_fd, NULL, NULL, NULL);
}
*/
import "C"
