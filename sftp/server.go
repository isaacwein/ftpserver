package sftp

import (
	"fmt"
	"github.com/pkg/sftp"
	"github.com/telebroad/ftpserver/filesystem"
	"github.com/telebroad/ftpserver/ftp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"os"
)

type Server struct {
	Addr       string
	logger     *slog.Logger
	fsFileRoot filesystem.FtpFS
	PrivateKey []byte
	sshConfig  *ssh.ServerConfig
	sftpServer *sftp.RequestServer
	sshServer  *ssh.ServerConn
	listener   net.Listener
	users      ftp.Users
}

func NewSFTPServer(logger slog.Logger, fs filesystem.FtpFS) (any, error) {

	s := &Server{
		Addr:       os.Getenv("SFTP_SERVER_ADDR"),
		logger:     logger.With("Module", "NewSFTPServer()"),
		fsFileRoot: fs,
	}
	pk, _, err := GeneratesRSAKeys(2048)
	if err != nil {
		return nil, fmt.Errorf("error generating RSA keys: %w", err)
	}
	s.PrivateKey = pk
	// Configure the SSH server settings.
	s.sshConfig = &ssh.ServerConfig{
		PasswordCallback: s.AuthHandler,
	}

	// Generate a new key pair for the server.
	privateKey, err := ssh.ParsePrivateKey(s.PrivateKey)
	if err != nil {
		s.logger.Error("Error parsing private key", "error", err)
		err = fmt.Errorf("error parsing private key: %w", err)
		return s, err
	}

	s.sshConfig.AddHostKey(privateKey)

	// Start the SSH server.
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		s.logger.Error("Failed to listen", "error", err)
		err = fmt.Errorf("failed to listen: %w", err)
		return s, err
	}

	s.logger.Info("Listening on " + s.Addr)

	for {
		// Accept incoming connections.
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept incoming connection (%v)\n", err)
			continue
		}

		// Handle each connection in a new goroutine.
		go s.sshHandler(conn)
	}
}

func (s *Server) Close() error {
	err := s.sftpServer.Close()
	if err != nil {
		return err
	}
	err = s.sshServer.Close()
	if err != nil {
		return err
	}
	err = s.listener.Close()
	return err
}

// AuthHandler is called by the SSH server when a client attempts to authenticate.
func (s *Server) AuthHandler(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
	s.logger.Debug("Login temp", "user", c.User())
	if _, err := s.users.Find(c.User(), string(pass), c.RemoteAddr().String()); err == nil {
		return nil, nil
	}

	return nil, fmt.Errorf("password rejected for %q", c.User())
}

func (s *Server) sshHandler(conn net.Conn) {
	defer conn.Close()

	// Upgrade the connection to an SSH connection.
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.sshConfig)
	if err != nil {
		s.logger.Error("Failed to handshake", "error", err)
		return
	}
	s.sshServer = sshConn

	s.logger.Info(
		"New SSH connection",
		"RemoteAddr", sshConn.RemoteAddr().String(),
		"ClientVersion", sshConn.ClientVersion(),
		"ServerVersion", sshConn.ServerVersion(),
		"ssh-User", sshConn.User(),
		"SessionID", sshConn.SessionID(),
	)
	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		// Channels have a type, depending on the application level protocol intended. In the case of an SFTP
		// server, we expect a channel type of "session". The SFTP server operates over a single channel.

		s.logger.Debug("Incoming channel", "channelType", newChannel.ChannelType())
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			fmt.Printf("Could not accept channel (%v)\n", err)
			return
		}

		// Start an SFTP session.
		go s.filterHandler(requests)

		FS := NewVirtualDirectory(s.fsFileRoot.(*filesystem.FtpLocalFS).FS, sftp.InMemHandler(), s.logger)
		s.sftpServer = sftp.NewRequestServer(channel, FS)

		if err := s.sftpServer.Serve(); err == io.EOF {
			s.sftpServer.Close()
			s.logger.Info("sftp client exited session.", "user", sshConn.User())

		} else if err != nil {
			s.logger.Error("sftp server completed with error", "error", err)
		}

	}
}

type virtualDirectory struct {
	handler sftp.Handlers
	logger  *slog.Logger
	fs      fs.FS
}

func NewVirtualDirectory(FS fs.FS, handler sftp.Handlers, logger *slog.Logger) sftp.Handlers {

	v := &virtualDirectory{
		handler: handler,
		logger:  logger,
		fs:      FS,
	}

	return sftp.Handlers{
		FileGet:  v,
		FilePut:  v,
		FileCmd:  v,
		FileList: v,
	}
}

func (v *virtualDirectory) Fileread(request *sftp.Request) (io.ReaderAt, error) {

	v.logger.Debug("Fileread",
		"request.Method:", request.Method, "\n",
		"request.Filepath:", request.Filepath, "\n",
		"request.Attrs:", request.Attrs, "\n",
		"request.Flags:", request.Flags, "\n",
		"request.Target:", request.Target)
	// Creating a pipe to connect your FtpFS Read method with the SFTP response
	fsFile, err := v.fs.Open(request.Filepath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	osFile, ok := fsFile.(*os.File)
	if !ok {
		return nil, fmt.Errorf("error casting file to os.File")
	}
	return osFile, nil
}

func (v *virtualDirectory) Filewrite(request *sftp.Request) (io.WriterAt, error) {

	v.logger.Debug("Filewrite",
		"request.Method:", request.Method, "\n",
		"request.Filepath:", request.Filepath, "\n",
		"request.Attrs:", request.Attrs, "\n",
		"request.Flags:", request.Flags, "\n",
		"request.Target:", request.Target,
	)

	return v.handler.FilePut.Filewrite(request)
}

func (v *virtualDirectory) Filecmd(request *sftp.Request) error {

	v.logger.Debug("Filecmd",
		"request.Method:", request.Method, "\n",
		"request.Filepath:", request.Filepath, "\n",
		"request.Attrs:", request.Attrs, "\n",
		"request.Flags:", request.Flags, "\n",
		"request.Target:", request.Target)
	return v.handler.FileCmd.Filecmd(request)
}

func (v *virtualDirectory) Filelist(request *sftp.Request) (sftp.ListerAt, error) {

	v.logger.Debug("Filelist",
		"request.Method:", request.Method, "\n",
		"request.Filepath:", request.Filepath, "\n",
		"request.Attrs:", request.Attrs, "\n",
		"request.Flags:", request.Flags, "\n",
		"request.Target:", request.Target)

	return v.handler.FileList.Filelist(request)
}

// Start an SFTP session.
func (s *Server) filterHandler(in <-chan *ssh.Request) {
	for req := range in {
		s.logger.Debug("Request", "type", req.Type, "payload", string(req.Payload))

		ok := false
		switch req.Type {
		case "subsystem":
			if string(req.Payload[4:]) == "sftp" {
				ok = true
			}
		}
		err := req.Reply(ok, nil)
		if err != nil {
			s.logger.Error("Failed to reply", "error", err)
			return
		}
	}
}
