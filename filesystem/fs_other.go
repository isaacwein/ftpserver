//go:build !linux && !darwin && !windows && !plan9

package filesystem

import (
	"fmt"
	"github.com/pkg/sftp"
	"runtime"
	"syscall"
)

// StatFS FileStatFS returns the file system status of the file system containing the file
func (FS *LocalFS) StatFS(path string) (*sftp.StatVFS, error) {
	return nil, fmt.Errorf("%w unsupported OS: %s", syscall.ENOTSUP, runtime.GOOS)
}
