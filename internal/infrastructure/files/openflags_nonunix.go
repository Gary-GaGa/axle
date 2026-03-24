//go:build !unix

package files

import "os"

const (
	ReadOnlyOpenFlags = os.O_RDONLY
	WriteOpenFlags    = os.O_WRONLY | os.O_CREATE
)
