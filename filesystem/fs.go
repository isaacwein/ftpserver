package filesystem

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

// FS is filesystem the interface that wraps the basic methods for a FTP,FTPS,SFTP file system
// The FTP server uses this interface to interact with the file system
// CheckDir checks if the given directory exists
// RootDir returns the Root directory of the file system
// Dir returns a list of files in the given directory
// Read reads the file and writes it to the given writer
// Create creates a new file with the given name and writes the data from the reader
// Remove removes the file/directory
// Rename renames the file/directory or moves it to a different directory
// Stat returns the file info
// SetStat changes the file info
// Lstat returns the file info without following the link
// Link creates a hard link pointing to a file.
// Symlink creates a symbolic link pointing to a file or directory.
type FS interface {

	// RootDir returns the Root directory of the file system
	RootDir() string
	// Dir returns a list of files in the given directory
	Dir(folderName string) ([]string, []os.FileInfo, error)
	// CheckDir checks if the given directory exists
	CheckDir(string) error
	// MakeDir creates a new directory with the given name
	MakeDir(folderName string) error
	// ReadFile reads the file and writes it to the given writer
	ReadFile(string, io.Writer) (int64, error)
	// WriteFile creates a new file with the given name and writes the data from the reader
	// filename is the name of the file to create
	// r is the reader that contains the data to write to the file
	// transferType is the transfer type "A" for ASCII or "I" for binary
	// appendOnly is true if the file should be opened in append mode not rewrite mode
	WriteFile(fileName string, r io.Reader, transferType string, appendOnly bool) error
	// Remove removes the file
	// fileName is the name of the file to remove
	Remove(fileName string) error
	// Rename renames the file/folder or moves it to a different directory
	Rename(original string, target string) error
	// ModifyTime changes the file modification time
	ModifyTime(string, string) error
	// Stat returns the file info without following the link
	Stat(fileName string) (string, fs.FileInfo, error)
	// SetStat changes the file info
	SetStat(fileName string, newPermissions uint32) error
	// Lstat returns the file info without following the link
	Lstat(fileName string) (string, fs.FileInfo, error)
	// Link creates a hard link pointing to a file.
	Link(fileName string, target string) error
	// Symlink creates a symbolic link pointing to a file or directory.
	Symlink(fileName string, target string) error
}

// NewFS implement the FS interface add support for the New 1.16 fs.FS interface
type NewFS interface {
	FS
	// GetFS returns the fs.FS object
	GetFS() fs.FS
}

// FSWithFile is the interface that wraps the basic methods for a SFTP file system
type FSWithFile interface {
	FS
	// File opens the file and returns a file object
	// fileName is the name of the file to open
	File(fileName string, access uint32) (*os.File, error)
}

// NewFSWithFile implement the FSWithFile interface add support for the New 1.16 fs.FS interface
type NewFSWithFile interface {
	NewFS
	FSWithFile
}

// Ensure that LocalFS implements the FtpFS interface
var _ NewFSWithFile = &LocalFS{}

// LocalFS is a local file system that implements the FtpFS interface
type LocalFS struct {
	FS          fs.FS
	localDir    string // local directory to serve as the ftp virtualRoot
	virtualRoot string // virtualRoot directory that the server is serving normally it is "/", if its deeper then add it to the system "dir/virtualRoot"
}

// RootDir returns the Root directory of the file system
func (FS *LocalFS) RootDir() string {
	return FS.virtualRoot
}

// CheckDir checks if the given directory exists
func (FS *LocalFS) CheckDir(dirName string) (err error) {

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

func (FS *LocalFS) GetFS() fs.FS {
	return FS.FS
}

// Dir returns a list of files in the given directory
func (FS *LocalFS) Dir(dirName string) ([]string, []os.FileInfo, error) {

	dirName, err := FS.cleanPath(dirName)
	if err != nil {
		return nil, nil, err
	}

	entries, err := fs.ReadDir(FS.FS, dirName)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading directory: %w", err)
	}

	lines := make([]string, len(entries))
	fileList := make([]os.FileInfo, len(entries))
	for i, entry := range entries {

		line, entry, err := FS.Stat(filepath.Join(dirName, entry.Name()))
		if err != nil {
			return nil, nil, err
		}
		lines[i] = line
		fileList[i] = entry
	}

	return lines, fileList, nil
}

// MakeDir creates a new directory with the given name
func (FS *LocalFS) MakeDir(folderName string) error {
	folderName, err := FS.cleanPath(folderName)
	if err != nil {
		return err
	}
	folderName = filepath.Join(FS.localDir, folderName)

	err = os.MkdirAll(folderName, 0777)
	if err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}
	return nil
}

// File reads the file at the given offset and writes it to the given writer
func (FS *LocalFS) File(fileName string, access uint32) (*os.File, error) {

	fileName, err := FS.cleanPath(fileName)
	if err != nil {
		return nil, err
	}

	// Open the file for reading
	fileName = filepath.Join(FS.localDir, fileName)

	file, err := os.OpenFile(fileName, int(access), 0666)
	if err != nil {
		return nil, fmt.Errorf("creating file error: %w", err)
	}

	return file, nil
}

// ReadFile reads the file and writes it to the given writer
func (FS *LocalFS) ReadFile(name string, w io.Writer) (int64, error) {
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

// WriteFile creates a new file with the given name and writes the data from the reader
func (FS *LocalFS) WriteFile(fileName string, r io.Reader, transferType string, appendOnly bool) error {
	fileName, err := FS.cleanPath(fileName)
	if err != nil {
		return err
	}
	fileName = filepath.Join(FS.localDir, fileName)
	access := 0
	if appendOnly {
		access = os.O_RDWR | os.O_CREATE | os.O_APPEND
	} else {
		access = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	}

	file, err := os.OpenFile(fileName, access, 0666)
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
func (FS *LocalFS) Remove(fileName string) (err error) {
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
func (FS *LocalFS) Rename(fileName, newName string) (err error) {
	fileName, err = FS.cleanPath(fileName)
	if err != nil {
		return err
	}

	newName, err = FS.cleanPath(newName)
	if err != nil {
		return err
	}

	fileName = filepath.Join(FS.localDir, fileName)
	newName = filepath.Join(FS.localDir, newName)

	fmt.Println("oldFile:", fileName, "newFileName:", newName)

	err = os.Rename(fileName, newName)
	if err != nil {
		return fmt.Errorf("error renaming file: %w", err)
	}
	return
}

// ModifyTime changes the file modification time
func (FS *LocalFS) ModifyTime(filePath string, newTime string) (err error) {
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

// Stat returns the file info
func (FS *LocalFS) Stat(fileName string) (string, fs.FileInfo, error) {

	fileName, err := FS.cleanPath(fileName)
	if err != nil {
		return "", nil, err
	}

	info, err := fs.Stat(FS.FS, fileName)
	if err != nil {
		return "", nil, fmt.Errorf("error getting file info: %w", err)
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
		info.Name()), info, nil
}
func (FS *LocalFS) SetStat(fileName string, newPermissions uint32) error {
	fileName, err := FS.cleanPath(fileName)
	if err != nil {
		return err
	}
	fileName = filepath.Join(FS.localDir, fileName)
	if newPermissions == 0 {
		return errors.New("invalid permissions")
	}
	newPermission := os.FileMode(newPermissions)
	err = os.Chmod(fileName, newPermission)
	if err != nil {
		return fmt.Errorf("error changing file permissions: %w", err)
	}
	return nil
}
func (FS *LocalFS) Lstat(fileName string) (string, fs.FileInfo, error) {
	fileName, err := FS.cleanPath(fileName)
	if err != nil {
		return "", nil, err
	}
	fileName = filepath.Join(FS.localDir, fileName)
	info, err := os.Lstat(fileName)
	if err != nil {
		return "", nil, fmt.Errorf("error getting file info: %w", err)
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
		info.Name()), info, nil
}

// Link creates a hard link pointing to a file.
func (FS *LocalFS) Link(fileName string, target string) (err error) {
	fileName, err = FS.cleanPath(fileName)
	if err != nil {
		return fmt.Errorf("error cleaning filname path: %w", err)
	}
	fileName = filepath.Join(FS.localDir, fileName)
	target, err = FS.cleanPath(target)
	if err != nil {
		return fmt.Errorf("error cleaning target path: %w", err)
	}
	target = filepath.Join(FS.localDir, target)
	return os.Link(target, fileName)
}

// Symlink creates a symbolic link pointing to a file or directory.
func (FS *LocalFS) Symlink(fileName string, target string) (err error) {
	fileName, err = FS.cleanPath(fileName)
	if err != nil {
		return fmt.Errorf("error cleaning filname path: %w", err)
	}
	fileName = filepath.Join(FS.localDir, fileName)
	target, err = FS.cleanPath(target)
	if err != nil {
		return err
	}
	target = filepath.Join(FS.localDir, target)
	return os.Symlink(target, fileName)
}

func NewLocalFS(localDir string) *LocalFS {
	ftpLocalFS := &LocalFS{
		localDir:    localDir,
		virtualRoot: "/",
		FS:          os.DirFS(localDir),
	}
	return ftpLocalFS
}

// securePath ensures that the given path is safe to use its dont allow to go outside the virtualRoot directory
func (FS *LocalFS) securePath(pathName string) (string, error) {
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
func (FS *LocalFS) cleanPath(pathName string) (string, error) {

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
