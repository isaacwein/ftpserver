package filesystem

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/sys/unix"
)

// StatFS FileStatFS returns the file system status of the file system containing the file
func (FS *LocalFS) StatFS(path string) (*sftp.StatVFS, error) {
	var stat unix.Statfs_t

	err := unix.Statfs(path, &stat)
	if err != nil {
		err = fmt.Errorf("error getting file system info: %w", err)
		return nil, err
	}

	sftpStatVFS := &sftp.StatVFS{
		Bsize:   uint64(stat.Bsize),
		Frsize:  uint64(stat.Bsize), // fragment size is a linux thing; use block size here
		Blocks:  stat.Blocks,
		Bfree:   stat.Bfree,
		Bavail:  stat.Bavail,
		Files:   stat.Files,
		Ffree:   stat.Ffree,
		Favail:  stat.Ffree,                                              // not sure how to calculate Favail
		Fsid:    uint64(stat.Fsid.Val[1])<<32 | uint64(stat.Fsid.Val[0]), // endianness?
		Flag:    uint64(stat.Flags),                                      // assuming POSIX?
		Namemax: 1024,                                                    // man 2 statfs shows: #define MAXPATHLEN      1024
	}

	return sftpStatVFS, nil
}
