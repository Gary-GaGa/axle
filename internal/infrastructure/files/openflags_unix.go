//go:build unix

package files

import (
	"os"

	"golang.org/x/sys/unix"
)

const (
	ReadOnlyOpenFlags = os.O_RDONLY | unix.O_NONBLOCK | unix.O_NOFOLLOW
	WriteOpenFlags    = os.O_WRONLY | os.O_CREATE | unix.O_NONBLOCK | unix.O_NOFOLLOW
)
