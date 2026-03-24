package app

import (
	"os"

	infrafiles "github.com/garyellow/axle/internal/infrastructure/files"
)

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	return infrafiles.WriteFileAtomic(path, data, perm)
}
