package sftp

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"github.com/telebroad/ftpserver/filesystem"
	"github.com/telebroad/ftpserver/tools"
	"golang.org/x/crypto/ssh"
	"io"
	"io/fs"
	"log/slog"
	"os"
)

type FileSys struct {
	fs     filesystem.FSWithFile
	logger *slog.Logger
}

func NewFileSys(FS filesystem.FSWithFile, logger *slog.Logger) sftp.Handlers {

	v := &FileSys{
		logger: logger,
		fs:     FS,
	}

	return sftp.Handlers{
		FileGet:  v,
		FilePut:  v,
		FileCmd:  v,
		FileList: v,
	}

}

func (s *Server) sftpHandler(channel ssh.Channel) {
	reader := bufio.NewReader(channel)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			s.Logger().Error("Failed to read command", "error", err)
			return
		}
		s.Logger().Debug("Command", "command", line)
	}

}

func (s *FileSys) Fileread(request *sftp.Request) (io.ReaderAt, error) {
	return s.OpenFile(request)
}
func (s *FileSys) Filewrite(request *sftp.Request) (io.WriterAt, error) {
	return s.OpenFile(request)
}
func (s *FileSys) OpenFile(request *sftp.Request) (sftp.WriterAtReaderAt, error) {

	s.logger.Debug("FileWrite",
		"request.Method:", request.Method,
		"request.Filepath:", request.Filepath,
		"request.Attrs:", tools.IsPrintable(request.Attrs),
		"request.Flags:", request.Flags,
		"request.Target:", request.Target,
	)
	if request.Method == "Put" {
		// Adjust the flags to ensure file creation
		request.Flags |= uint32(os.O_CREATE)
	}
	file, err := s.fs.File(request.Filepath, request.Flags)

	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	return file, nil
}
func (s *FileSys) Filecmd(request *sftp.Request) error {
	s.logger.Debug("Filecmd",
		"request.Method:", request.Method,
		"request.Filepath:", request.Filepath,
		"request.Attrs:", tools.IsPrintable(request.Attrs),
		"request.Flags:", request.Flags,
		"request.Target:", request.Target,
	)
	switch request.Method {
	case "Setstat", "chmod", "chown", "chgrp":

		err := s.fs.SetStat(request.Filepath, request.Attributes().FileMode())
		if err != nil {
			return err
		}
		return nil

	case "Rename":
		// SFTP-v2: "It is an error if there already exists a file with the name specified by newpath."
		// This varies from the POSIX specification, which allows limited replacement of target files.
		return s.PosixRename(request)

	case "Rmdir":

		err := s.fs.CheckDir(request.Filepath)
		if err != nil {
			return err
		}

		return s.fs.Remove(request.Filepath)

	case "Remove":
		// IEEE 1003.1 remove explicitly can unlink files and remove empty directories.
		// We use instead here the semantics of unlink, which is allowed to be restricted against directories.
		return s.fs.Remove(request.Filepath)

	case "Mkdir":

		return s.fs.MakeDir(request.Filepath)

	case "Link":
		return s.fs.Link(request.Filepath, request.Target)

	case "Symlink":
		// NOTE: r.Filepath is the target, and r.Target is the linkpath.
		return s.fs.Symlink(request.Filepath, request.Target)
	}

	return errors.New("unsupported")
}
func (s *FileSys) PosixRename(request *sftp.Request) error {
	s.logger.Debug("Filecmd",
		"request.Method:", request.Method,
		"request.Filepath:", request.Filepath,
		"request.Attrs:", tools.IsPrintable(request.Attrs),
		"request.Flags:", request.Flags,
		"request.Target:", request.Target,
	)
	_, _, err := s.fs.Stat(request.Target)
	if err == nil {
		return fs.ErrExist
	}

	return s.fs.Rename(request.Filepath, request.Target)
}

func (s *FileSys) StatVFS(request *sftp.Request) (*sftp.StatVFS, error) {
	s.logger.Debug("Filecmd",
		"request.Method:", request.Method,
		"request.Filepath:", request.Filepath,
		"request.Attrs:", tools.IsPrintable(request.Attrs),
		"request.Flags:", request.Flags,
		"request.Target:", request.Target,
	)

	return s.fs.StatFS(request.Filepath)
}

type ListerAt []os.FileInfo

// ListAt Modeled after strings.Reader's ReadAt() implementation
func (f ListerAt) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	var n int
	if offset >= int64(len(f)) {
		return 0, io.EOF
	}
	n = copy(ls, f[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}

func (s *FileSys) Filelist(request *sftp.Request) (sftp.ListerAt, error) {
	s.logger.Debug("Filelist",
		"request.Method:", request.Method,
		"request.Filepath:", request.Filepath,
		"request.Attrs:", tools.IsPrintable(request.Attrs),
		"request.Flags:", request.Flags,
		"request.Target:", request.Target,
	)
	var entries []os.FileInfo
	var err error
	switch request.Method {
	case "List":
		_, entries, err = s.fs.Dir(request.Filepath)
		if err != nil {
			return nil, fmt.Errorf("fileList error: %w", err)
		}
	case "Stat":
		_, entrie, err := s.fs.Stat(request.Filepath)
		if err != nil {
			return nil, fmt.Errorf("fileStat error: %w", err)
		}
		entries = []os.FileInfo{entrie}
	case "Lstat":
		_, entrie, err := s.fs.Lstat(request.Filepath)
		if err != nil {
			return nil, fmt.Errorf("lstat error: %w", err)
		}
		entries = []os.FileInfo{entrie}

	}

	return ListerAt(entries), nil
}
