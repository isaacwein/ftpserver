package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FtpFS interface {
	CheckDir(string) error
	RootDir() string
	Dir(string) ([]string, error)
	File()
	Open()
	Create(string, io.Reader, string) error
	Remove()
	Rename()
	Stat()
	Lstat()
}
type FtpLocalFS struct {
	FS          fs.FS
	localDir    string // local directory to serve as the ftp virtualRoot
	virtualRoot string // virtualRoot directory that the server is serving normally it is "/", if its deeper then add it to the system "dir/virtualRoot"
}

func (FS *FtpLocalFS) RootDir() string {
	return FS.virtualRoot
}

func (FS *FtpLocalFS) CheckDir(name string) (err error) {
	fmt.Println("*FtpLocalFS.CheckDir", name, filepath.Clean(name))
	if len(name) == 0 || name == "/" {
		name = "."
	}
	_, err = fs.ReadDir(FS.FS, name)
	return
}

func (FS *FtpLocalFS) Dir(name string) ([]string, error) {

	if name == "" || name == "/" {
		name = "."
	} else if (name[0]) == '/' {
		name = name[1:]
	}
	fmt.Println("*FtpLocalFS.Dir", name)
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
func (FS *FtpLocalFS) Create(fileName string, w io.Reader, transferType string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("creating file error: %w", err)
	}
	defer file.Close()

	if transferType == "I" { // Binary mode
		_, err = io.Copy(file, w) // Directly copy data without conversion
	} else if transferType == "A" { // ASCII mode
		// Use a bufio.Scanner to handle line endings conversion
		scanner := bufio.NewScanner(w)
		for scanner.Scan() {
			line := scanner.Text()
			_, err = fmt.Fprintln(file, line) // Append a newline appropriate for the server's OS
		}
	} else {
		return fmt.Errorf("unsupported transfer type: %s, only type 'A' (text) or type 'I' (binary)", transferType)
	}

	if err != nil {
		return fmt.Errorf("writing file error: %w", err)
	}
	return nil
}
func (FS *FtpLocalFS) Remove() {

}
func (FS *FtpLocalFS) Rename() {

}
func (FS *FtpLocalFS) Stat() {

}
func (FS *FtpLocalFS) Lstat() {

}

func NewFtpLocalFS(localDir, virtualRoot string) *FtpLocalFS {
	virtualRoot = filepath.Clean(virtualRoot)
	ftpLocalFS := &FtpLocalFS{
		localDir:    localDir,
		virtualRoot: virtualRoot,
		FS:          os.DirFS(localDir),
	}
	return ftpLocalFS
}

// cleanPath ensures that the given path is safe to use
func (FS *FtpLocalFS) cleanPath(name string) (string, error) {
	cleaned := filepath.Clean(name)
	relPath, err := filepath.Rel(FS.virtualRoot, filepath.Join(FS.virtualRoot, cleaned))
	if err != nil {
		return "", err
	}

	if relPath == "." || relPath == FS.virtualRoot {
		return cleaned, nil
	}

	if !strings.HasPrefix(relPath, "..") && !os.IsPathSeparator(relPath[0]) {
		return filepath.Join(FS.virtualRoot, cleaned), nil
	}

	return "", errors.New("access denied: path is outside the virtualRoot directory")
}
