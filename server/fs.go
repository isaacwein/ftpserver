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

// FtpFS is the interface that wraps the basic methods for a FTP file system
// The FTP server uses this interface to interact with the file system
// CheckDir checks if the given directory exists
// RootDir returns the root directory of the file system
// Dir returns a list of files in the given directory
// Read reads the file and writes it to the given writer
// Create creates a new file with the given name and writes the data from the reader
// Remove removes the file/directory
// Rename renames the file/directory or moves it to a different directory
// Stat returns the file info
type FtpFS interface {
	CheckDir(string) error
	RootDir() string
	Dir(string) ([]string, error)
	Read(string, io.Writer) (int64, error)
	Create(string, io.Reader, string) error
	Remove(string) error
	Rename(string, string) error
	Stat(string) (string, error)
	ModifyTime(string, string) error
}

// Ensure that FtpLocalFS implements the FtpFS interface
var _ FtpFS = &FtpLocalFS{}

// FtpLocalFS is a local file system that implements the FtpFS interface
type FtpLocalFS struct {
	FS          fs.FS
	localDir    string // local directory to serve as the ftp virtualRoot
	virtualRoot string // virtualRoot directory that the server is serving normally it is "/", if its deeper then add it to the system "dir/virtualRoot"
}

// RootDir returns the root directory of the file system
func (FS *FtpLocalFS) RootDir() string {
	return FS.virtualRoot
}

// CheckDir checks if the given directory exists
func (FS *FtpLocalFS) CheckDir(dirName string) (err error) {

	dirName, err = FS.cleanPath(dirName)
	if err != nil {
		return err
	}

	_, err = fs.ReadDir(FS.FS, dirName)
	if err != nil {
		return fmt.Errorf("error checking directory: %w", err)
	}
	return
}

// Dir returns a list of files in the given directory
func (FS *FtpLocalFS) Dir(dirName string) ([]string, error) {

	dirName, err := FS.cleanPath(dirName)
	if err != nil {
		return nil, err
	}

	entries, err := fs.ReadDir(FS.FS, dirName)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	var lines []string
	for _, entry := range entries {

		line, err := FS.Stat(filepath.Join(dirName, entry.Name()))
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}

	return lines, nil
}

// Stat returns the file info
func (FS *FtpLocalFS) Stat(fileName string) (string, error) {

	fileName, err := FS.cleanPath(fileName)
	if err != nil {
		return "", err
	}

	info, err := fs.Stat(FS.FS, fileName)
	if err != nil {
		return "", fmt.Errorf("error getting file info: %w", err)
	}
	fileType := "file"
	if info.IsDir() {
		fileType = "dir"
	}

	mode := info.Mode()
	size := info.Size()
	modTime := info.ModTime().UTC().Format("20060102150405")
	// FTP format: permissions, number of links, owner, group, size, modification time, name
	return fmt.Sprintf("Type=%s;Size=%d;Modify=%s;Perm=%s;UNIX.ownername=%s;UNIX.groupname=%s; %s",
		fileType, size, modTime, mode.String(), "owner", "group",
		info.Name()), nil
}

// Read reads the file and writes it to the given writer
func (FS *FtpLocalFS) Read(name string, w io.Writer) (int64, error) {
	// Open the file for reading
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	open, err := FS.FS.Open(name)
	if err != nil {
		return 0, fmt.Errorf("error opening file: %w", err)
	}
	defer open.Close()
	n, err := io.Copy(w, open)
	if err != nil {
		return n, fmt.Errorf("error reading file: %w", err)
	}
	return n, nil
}

// Create creates a new file with the given name and writes the data from the reader
func (FS *FtpLocalFS) Create(fileName string, r io.Reader, transferType string) error {

	fileName = filepath.Join(FS.localDir, fileName)
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("creating file error: %w", err)
	}
	defer file.Close()

	if transferType == "I" { // Binary mode
		_, err = io.Copy(file, r) // Directly copy data without conversion
	} else if transferType == "A" { // ASCII mode
		// Use a bufio.Scanner to handle line endings conversion
		scanner := bufio.NewScanner(r)
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
	_ = file.Close()
	if err != nil {
		return fmt.Errorf("closing and saving file error: %w", err)
	}
	return nil
}

// Remove removes the file
func (FS *FtpLocalFS) Remove(fileName string) (err error) {
	fileName, err = FS.cleanPath(fileName)
	if err != nil {
		return err
	}

	fileName = filepath.Join(FS.localDir, fileName)

	err = os.Remove(fileName)
	if err != nil {
		return fmt.Errorf("error removing file: %w", err)
	}
	return
}

// Rename renames the file or moves it to a different directory
func (FS *FtpLocalFS) Rename(fileName, newName string) (err error) {
	fileName, err = FS.cleanPath(fileName)
	if err != nil {
		return err
	}
	_, err = os.Stat(fileName)
	if err != nil {
		// Handle error, for example, file does not exist.
		return fmt.Errorf("error getting file info: %w", err)
	}
	newName, err = FS.cleanPath(newName)
	if err != nil {
		return err
	}
	fileName = filepath.Join(FS.localDir, fileName)
	err = os.Rename(fileName, newName)
	if err != nil {
		return fmt.Errorf("error renaming file: %w", err)
	}
	return
}

// ModifyTime changes the file modification time
func (FS *FtpLocalFS) ModifyTime(filePath string, newTime string) (err error) {
	filePath, err = FS.cleanPath(filePath)
	if err != nil {
		return err
	}
	newTimeP, err := time.Parse("20060102150405", newTime)
	if err != nil {
		return fmt.Errorf("501 Invalid time format got '%s' expected 'YYYYMMDDHHMMSS'", newTime)
	}
	filePath = filepath.Join(FS.localDir, filePath)
	_, err = os.Stat(filePath)
	if err != nil {
		// Handle error, for example, file does not exist.
		return fmt.Errorf("error getting file info: %w", err)
	}
	// Change the file modification time
	err = os.Chtimes(filePath, newTimeP, newTimeP)
	if err != nil {
		return fmt.Errorf("error changing file modification time: %w", err)
	}
	return
}

// Lstat returns the file info without following the link
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

// securePath ensures that the given path is safe to use its dont allow to go outside the virtualRoot directory
func (FS *FtpLocalFS) securePath(pathName string) (string, error) {
	cleaned := filepath.Clean(pathName)
	relPath, err := filepath.Rel(FS.virtualRoot, filepath.Join(FS.virtualRoot, cleaned))
	if err != nil {
		return "", fmt.Errorf("error cleaning path: %w", err)

	}

	pathName = filepath.Join(FS.virtualRoot, cleaned)

	if relPath == "." || relPath == FS.virtualRoot {
		return cleaned, nil
	}

	if !strings.HasPrefix(relPath, "..") && !os.IsPathSeparator(relPath[0]) {
		return pathName, nil
	}

	return "", errors.New("access denied: path is outside the virtualRoot directory")
}

// cleanPath call securePath and then clean the path to be used
func (FS *FtpLocalFS) cleanPath(pathName string) (string, error) {

	pathName, err := FS.securePath(pathName)
	if err != nil {
		return "", err
	}
	if pathName == "" || pathName == "/" {
		pathName = "."
	} else if (pathName[0]) == '/' {
		pathName = pathName[1:]
	}

	return pathName, nil

}
