//go:build windows

package filesystem

import (
	"github.com/pkg/sftp"
	"golang.org/x/sys/windows"
	"syscall"
)

func (FS *LocalFS) StatFS(path string) (*sftp.StatVFS, error) {
	statvfs := &sftp.StatVFS{}

	// Get disk free space
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	drive, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	err = windows.GetDiskFreeSpaceEx(drive, &freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	if err != nil {
		return nil, err
	}

	// Calculating block size based on total and free bytes might not be accurate
	// Since we don't have direct access to bytes per sector and sectors per cluster here
	// We'll use a default or make a separate syscall (not shown here) for those values

	bsize := uint64(4096) // Assuming a default block size; consider updating based on actual cluster size if necessary
	statvfs.Bsize = bsize
	statvfs.Frsize = bsize
	statvfs.Blocks = totalNumberOfBytes / bsize
	statvfs.Bfree = totalNumberOfFreeBytes / bsize
	statvfs.Bavail = freeBytesAvailable / bsize

	// Set Namemax to 255, typical for Windows filesystems
	statvfs.Namemax = 255

	// ID, Files, Ffree, Favail, and Flag are not directly available or not applicable on Windows

	return statvfs, nil
}
