package sftp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"github.com/telebroad/fileserver/filesystem"
	"github.com/telebroad/fileserver/tools"
	"golang.org/x/crypto/ssh"
	"io"
	"io/fs"
	"log/slog"
	"os"
)

type Sessions struct {
	fs       filesystem.FSWithReadWriteAt
	logger   *slog.Logger
	ctx      context.Context
	cancel   context.CancelCauseFunc
	UserInfo ssh.ConnMetadata
}

func NewFileSys(Sessions *Sessions) sftp.Handlers {

	v := Sessions
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

func (s *Sessions) Fileread(request *sftp.Request) (io.ReaderAt, error) {

	s.logger.Debug("FileWrite",
		"request.Method:", request.Method,
		"request.Filepath:", request.Filepath,
		"request.Attrs:", tools.IsPrintable(request.Attrs),
		"request.Flags:", request.Flags,
		"request.Target:", request.Target,
	)
	file, err := s.fs.FileRead(request.Filepath, os.O_RDONLY)

	if err != nil {
		s.logger.Error("error opening file", "error", err)
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	return file, nil
}

func (s *Sessions) Filewrite(request *sftp.Request) (io.WriterAt, error) {

	s.logger.Debug("FileWrite",
		"request.Method:", request.Method,
		"request.Filepath:", request.Filepath,
		"request.Attrs:", tools.IsPrintable(request.Attrs),
		"request.Flags:", request.Flags,
		"request.Target:", request.Target,
	)

	file, err := s.fs.FileWrite(request.Filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC)

	if err != nil {
		s.logger.Error("error opening file", "error", err)
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return file, nil
}

func (s *Sessions) Filecmd(request *sftp.Request) error {
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
func (s *Sessions) PosixRename(request *sftp.Request) error {
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

func (s *Sessions) StatVFS(request *sftp.Request) (*sftp.StatVFS, error) {
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

func (s *Sessions) Filelist(request *sftp.Request) (sftp.ListerAt, error) {
	s.logger.Debug("Filelist",
		"request.Method:", request.Method,
		"request.Filepath:", request.Filepath,
		"request.Attrs:", tools.IsPrintable(request.Attrs),
		"request.Flags:", request.Flags,
		"request.Target:", request.Target,
	)

	var entry fs.FileInfo
	var entries []os.FileInfo
	var err error

	switch request.Method {
	case "List":
		_, entries, err = s.fs.Dir(request.Filepath)
		if err != nil {
			s.logger.Error("Filelist error", "error", err)
			err = fmt.Errorf("fileList error: %w", err)
			return nil, err
		}
	case "Stat":
		_, entry, err = s.fs.Stat(request.Filepath)
		if err != nil {
			s.logger.Error("fileStat error", "error", err)
			err = fmt.Errorf("fileStat error: %w", err)
			return nil, err
		}
		entries = []os.FileInfo{entry}
	case "Lstat":
		_, entry, err = s.fs.Lstat(request.Filepath)
		if err != nil {
			s.logger.Error("lstat error", "error", err)
			err = fmt.Errorf("lstat error: %w", err)
			return nil, err
		}
		entries = []os.FileInfo{entry}
	}

	return ListerAt(entries), nil
}
