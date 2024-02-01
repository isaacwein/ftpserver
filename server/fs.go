package server

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FtpFS interface {
	CheckDir(name string) (err error)
	RootDir() string
	Dir(string) ([]string, error)
	File()
	Open()
	Create()
	Remove()
	Rename()
	Stat()
	Lstat()
}
type FtpLocalFS struct {
	FS       fs.FS
	localDir string // local directory to serve as the ftp root
	root     string // root directory that the server is serving normally it is "/", if its deeper then add it to the system "dir/root"
}

func (FS *FtpLocalFS) RootDir() string {
	return FS.root
}

func (FS *FtpLocalFS) CheckDir(name string) (err error) {
	_, err = fs.ReadDir(FS.FS, name)

	return
}

func (FS *FtpLocalFS) Dir(name string) ([]string, error) {
	entries, err := fs.ReadDir(FS.FS, name)
	if err != nil {
		return nil, err
	}

	var lines []string
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}

		mode := info.Mode()
		size := info.Size()
		modTime := info.ModTime()

		// FTP format: permissions, number of links, owner, group, size, modification time, name
		line := fmt.Sprintf("%s %d %s %s %d %s %s",
			mode.String(), 1, "owner", "group", size,
			modTime.Format(time.Stamp), entry.Name())
		lines = append(lines, line)
	}

	return lines, nil
}

func (FS *FtpLocalFS) File() {

}

func (FS *FtpLocalFS) Open() {

}
func (FS *FtpLocalFS) Create() {

}
func (FS *FtpLocalFS) Remove() {

}
func (FS *FtpLocalFS) Rename() {

}
func (FS *FtpLocalFS) Stat() {

}
func (FS *FtpLocalFS) Lstat() {

}

func NewFtpLocalFS(localDir, root string) *FtpLocalFS {
	root = filepath.Clean(root)
	return &FtpLocalFS{
		localDir: localDir,
		root:     root,
		FS:       os.DirFS(root),
	}

}

// cleanPath ensures that the given path is safe to use
func (FS *FtpLocalFS) cleanPath(name string) (string, error) {
	cleaned := filepath.Clean(name)
	relPath, err := filepath.Rel(FS.root, filepath.Join(FS.root, cleaned))
	if err != nil {
		return "", err
	}

	if relPath == "." || relPath == FS.root {
		return cleaned, nil
	}

	if !strings.HasPrefix(relPath, "..") && !os.IsPathSeparator(relPath[0]) {
		return filepath.Join(FS.root, cleaned), nil
	}

	return "", errors.New("access denied: path is outside the root directory")
}
