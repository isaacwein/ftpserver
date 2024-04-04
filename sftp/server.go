package sftp

import (
	"context"
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"github.com/telebroad/fileserver/filesystem"
	"github.com/telebroad/fileserver/tools"
	"golang.org/x/crypto/ssh"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"
)

type Server struct {
	Addr             string
	logger           *slog.Logger
	fsFileRoot       filesystem.FSWithReadWriteAt
	privateKey       []byte
	privateKeySigner ssh.Signer
	sftpServer       *sftp.RequestServer
	sshServerConn    map[net.Conn]*Sessions
	listener         net.Listener
	users            Users
}

// Users is the interface to find a user by username and password and return it
type Users interface {
	// FindUser returns a user by username and password, if the user is not found it returns an error
	FindUser(ctx context.Context, username, password, ipaddr string) (any, error)
}

func NewSFTPServer(addr string, fs filesystem.FSWithReadWriteAt, users Users) *Server {

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
	s.privateKey = pk
}

// GetPrivateKey returns the private key for the server.
func (s *Server) GetPrivateKey() []byte {
	return s.privateKey
}

func (s *Server) SetPrivateKeyFile(pk string) error {
	file, err := os.ReadFile(pk)
	if err != nil {
		err = fmt.Errorf("error reading private key file: %w", err)
		return err
	}

	s.privateKey = file
	return nil
}

func (s *Server) ListenAndServe() error {
	s.sshServerConn = make(map[net.Conn]*Sessions)
	// Generate a new key pair if not set.
	if s.privateKey == nil {
		pk, _, err := GeneratesED25519Keys()
		if err != nil {
			return fmt.Errorf("error generating RSA keys: %w", err)
		}
		s.privateKey = pk
	}

	// Generate a new key pair for the server.
	privateKey, err := ssh.ParsePrivateKey(s.privateKey)
	if err != nil {
		s.Logger().Error("Error parsing private key", "error", err)
		err = fmt.Errorf("error parsing private key: %w", err)
		return err
	}

	s.privateKeySigner = privateKey

	// Start the SSH server.
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		s.Logger().Error("Failed to listen", "error", err)
		err = fmt.Errorf("failed to listen: %w", err)
		return err
	}

	s.Logger().Debug("Listening on " + s.Addr)

	for {
		// Accept incoming connections.
		conn, err := listener.Accept()
		if err != nil {
			s.Logger().Error("Failed to accept incoming connection", "error", err)
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
	wg := sync.WaitGroup{}
	for conn, ctx := range s.sshServerConn {
		wg.Add(1)
		go func(conn net.Conn, ctx *Sessions) {
			conn.Close()
			ctx.cancel(errors.New("server closed"))
			delete(s.sshServerConn, conn)
			wg.Done()
		}(conn, ctx)
	}
	wg.Wait()
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
func (s *Server) AuthHandler(conn net.Conn) func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	return func(m ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {

		session, ok := s.sshServerConn[conn]
		if !ok {
			s.Logger().Error("Session not found", "user", m.User())
			return nil, fmt.Errorf("session not found")
		}
		session.logger = session.logger.With("user", m.User())
		session.UserInfo = m
		ctx, cancel := context.WithTimeoutCause(session.ctx, 5*time.Second, fmt.Errorf("login timeout"))
		defer cancel()
		s.Logger().Debug("Login temp", "user", m.User())
		_, err := s.users.FindUser(ctx, m.User(), string(pass), m.RemoteAddr().String())
		if err == nil {
			session.logger = session.logger.With("User authenticated", true)
			return nil, nil
		}

		return nil, fmt.Errorf("password rejected for %q", m.User())
	}
}

func (s *Server) sshHandler(conn net.Conn) {
	defer conn.Close()
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	session := &Sessions{ctx: ctx, cancel: cancel, logger: s.Logger(), fs: s.fsFileRoot}
	s.sshServerConn[conn] = session
	defer delete(s.sshServerConn, conn)
	sshCfg := &ssh.ServerConfig{
		PasswordCallback: s.AuthHandler(conn),
	}
	sshCfg.AddHostKey(s.privateKeySigner)
	// Upgrade the connection to an SSH connection.
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, sshCfg)
	if err != nil {
		s.Logger().Error("Failed to handshake", "error", err)
		return
	}
	defer sshConn.Close()

	s.Logger().Debug(
		"New SSH connection",
		"RemoteAddr", sshConn.RemoteAddr().String(),
		"ClientVersion", string(sshConn.ClientVersion()),
		"ServerVersion", string(sshConn.ServerVersion()),
		"ssh-User", sshConn.User(),
		"SessionID", tools.IsPrintable(sshConn.SessionID()),
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
			s.Logger().Error("Could not accept channel", "error", err)
			return
		}

		// Start an SFTP session.
		go s.filterHandler(requests)

		serverOptions := []sftp.RequestServerOption{}

		FS := NewFileSys(session)
		s.sftpServer = sftp.NewRequestServer(channel, FS, serverOptions...)
		//s.sftpServer, err = sftp.NewServer(channel, serverOptions...)

		if err := s.sftpServer.Serve(); err == io.EOF {
			s.sftpServer.Close()
			s.Logger().Debug("sftp client exited session.", "user", sshConn.User())
		} else if err != nil {
			s.Logger().Error("sftp server completed with error", "error", err)
		}

	}
}

// Start an SFTP session.
func (s *Server) filterHandler(in <-chan *ssh.Request) {
	for req := range in {
		s.Logger().Debug("Request", "type", req.Type, "payload", tools.IsPrintable(string(req.Payload)))

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
