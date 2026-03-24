//go:build !unix

package files

import "os"

// HasMultipleLinks reports whether a file is backed by more than one hard link.
func HasMultipleLinks(info os.FileInfo) bool {
	return false
}
