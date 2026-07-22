//go:build linux

package sandboxmeasure

import (
	"os"

	"golang.org/x/sys/unix"
)

func openTokenNoFollow(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	// Generic name only — never embed the sensitive path in the File for error surfaces.
	return os.NewFile(uintptr(fd), "token"), nil
}
