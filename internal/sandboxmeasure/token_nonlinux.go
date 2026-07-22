//go:build !linux

package sandboxmeasure

import (
	"os"

	"golang.org/x/sys/unix"
)

// openTokenNoFollow for darwin/other unix test hosts: O_NOFOLLOW when the kernel supports it.
func openTokenNoFollow(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "token"), nil
}
