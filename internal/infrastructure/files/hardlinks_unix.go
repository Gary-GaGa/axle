//go:build unix

package files

import (
	"os"
	"syscall"
)

// HasMultipleLinks reports whether a file is backed by more than one hard link.
func HasMultipleLinks(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1
}
