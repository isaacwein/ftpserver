package sftp

import (
	"fmt"
	"github.com/pkg/sftp"
	"github.com/telebroad/ftpserver/filesystem"
	"github.com/telebroad/ftpserver/ftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log/slog"
	"net"
	"os"
	"time"
)

type Server struct {
	Addr       string
	logger     *slog.Logger
	fsFileRoot filesystem.FSWithFile
	PrivateKey []byte
	sshConfig  *ssh.ServerConfig
	sftpServer *sftp.RequestServer
	sshServer  *ssh.ServerConn
	listener   net.Listener
	users      ftp.Users
}

func NewSFTPServer(addr string, fs filesystem.FSWithFile, users ftp.Users) *Server {

	s := &Server{
		Addr:       addr,
		fsFileRoot: fs,
		users:      users,
	}

	return s
}

// SetPrivateKey sets the private key for the server.
// if not called the server will generate a new key
func (s *Server) SetPrivateKey(pk []byte) {
	s.PrivateKey = pk
}

func (s *Server) SetPrivateKeyFile(pk string) error {
	file, err := os.ReadFile(pk)
	if err != nil {
		err = fmt.Errorf("error reading private key file: %w", err)
		return err
	}

	s.PrivateKey = file
	return nil
}

func (s *Server) ListenAndServe() error {

	// Generate a new key pair if not set.
	if s.PrivateKey == nil {
		pk, _, err := GeneratesRSAKeys(2048)
		if err != nil {
			return fmt.Errorf("error generating RSA keys: %w", err)
		}
		s.PrivateKey = pk
	}

	// Configure the SSH server settings.
	s.sshConfig = &ssh.ServerConfig{
		PasswordCallback: s.AuthHandler,
	}

	// Generate a new key pair for the server.
	privateKey, err := ssh.ParsePrivateKey(s.PrivateKey)
	if err != nil {
		s.Logger().Error("Error parsing private key", "error", err)
		err = fmt.Errorf("error parsing private key: %w", err)
		return err
	}

	s.sshConfig.AddHostKey(privateKey)

	// Start the SSH server.
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		s.Logger().Error("Failed to listen", "error", err)
		err = fmt.Errorf("failed to listen: %w", err)
		return err
	}

	s.Logger().Info("Listening on " + s.Addr)

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

// TryListenAndServe tries to start the FTP server if there isn't an error after a certain time it returns nil
func (s *Server) TryListenAndServe(d time.Duration) (err error) {
	errC := make(chan error)

	go func() {
		err = s.ListenAndServe()
		if err != nil {
			errC <- err
		}
	}()

	select {
	case err = <-errC:
		return err
	case <-time.After(d):
		return nil
	}
}

// Close closes the server.
func (s *Server) Close() {
	s.sftpServer.Close()
	s.sshServer.Close()
	s.listener.Close()
	return
}

// SetLogger sets the logger for the server.
func (s *Server) SetLogger(l *slog.Logger) {
	s.logger = l
}

// Logger returns the logger for the server.
func (s *Server) Logger() *slog.Logger {
	if s.logger == nil {
		s.logger = slog.Default()
	}
	return s.logger.With("module", "sftp-server")
}

// AuthHandler is called by the SSH server when a client attempts to authenticate.
func (s *Server) AuthHandler(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
	s.Logger().Debug("Login temp", "user", c.User())
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
		s.Logger().Error("Failed to handshake", "error", err)
		return
	}
	s.sshServer = sshConn

	s.Logger().Info(
		"New SSH connection",
		"RemoteAddr", sshConn.RemoteAddr().String(),
		"ClientVersion", string(sshConn.ClientVersion()),
		"ServerVersion", string(sshConn.ServerVersion()),
		"ssh-User", sshConn.User(),
		"SessionID", string(sshConn.SessionID()),
	)
	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		// Channels have a type, depending on the application level protocol intended. In the case of an SFTP
		// server, we expect a channel type of "session". The SFTP server operates over a single channel.

		s.Logger().Debug("Incoming channel", "channelType", newChannel.ChannelType())
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

		serverOptions := []sftp.RequestServerOption{
			func(s *sftp.RequestServer) {
				// logs all sftp commands to stdout
				s.Reader = io.TeeReader(s.Reader, os.Stdout)
			},
		}

		FS := NewFileSys(s.fsFileRoot, s.logger)
		s.sftpServer = sftp.NewRequestServer(channel, FS, serverOptions...)
		//s.sftpServer, err = sftp.NewServer(channel, serverOptions...)

		if err := s.sftpServer.Serve(); err == io.EOF {
			s.sftpServer.Close()
			s.Logger().Info("sftp client exited session.", "user", sshConn.User())

		} else if err != nil {
			s.Logger().Error("sftp server completed with error", "error", err)
		}

	}
}

// Start an SFTP session.
func (s *Server) filterHandler(in <-chan *ssh.Request) {
	for req := range in {
		s.Logger().Debug("Request", "type", req.Type, "payload", string(req.Payload))

		ok := false
		switch req.Type {
		case "subsystem":
			if string(req.Payload[4:]) == "sftp" {
				ok = true
			}
		}
		err := req.Reply(ok, nil)
		if err != nil {
			s.Logger().Error("Failed to reply", "error", err)
			return
		}
	}
}
