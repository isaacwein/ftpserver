package ftp

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/telebroad/ftpserver/filesystem"
	"log/slog"
	"net"
	"net/netip"
	"time"
)

type FTPServerTransferType string

const (
	typeA FTPServerTransferType = "A"
	typeI FTPServerTransferType = "I"
)

// Users is the interface to find a user by username and password and return it
type Users interface {
	// Find returns a user by username and password, if the user is not found it returns an error
	Find(username, password, ipaddr string) (any, error)
}

type Server struct {
	// listener is the server listener
	listener net.Listener
	// Addr is the server address
	Addr string
	// supportsTLS is a flag to indicate if the server supports TLS
	FsHandler filesystem.FS
	// Root is the server root directory
	Root string
	//  sessionManager is the server session manager
	sessionManager *SessionManager
	// users is the server users
	users Users
	// WelcomeMessage is the server welcome message
	WelcomeMessage string
	// PublicServerIPv4 is the server public IPv4 address for passive mode
	PublicServerIPv4 [4]byte
	// Type is the server transfer type
	Type FTPServerTransferType
	// PasvMaxPort is the server passive mode max port
	PasvMaxPort int
	// PasvMinPort is the server passive mode min port
	PasvMinPort int
	//  TLS is the server TLS configuration
	TLS *tls.Config
	// TLSe is the server TLS configuration for upgrade existing FTP connection
	TLSe *tls.Config
	// Closer is the server closer channel on close the channel will return the error
	Closer chan error
	ctx    context.Context
	cancel context.CancelCauseFunc
	logger *slog.Logger
}

// NewServer creates a new FTP server
func NewServer(addr string, fsHandler filesystem.FS, users Users) (*Server, error) {
	s := &Server{
		Addr:           addr,
		FsHandler:      fsHandler,
		sessionManager: NewSessionManager(),
		users:          users,
		Root:           fsHandler.RootDir(),
		WelcomeMessage: "Welcome to My FTP Server",
		PasvMaxPort:    30000,
		PasvMinPort:    30100,
		Closer:         make(chan error),
	}
	s.ctx, s.cancel = context.WithCancelCause(context.Background())
	return s, nil
}

// WithContext sets the server context
func (s *Server) WithContext(ctx context.Context) *Server {
	s.ctx, s.cancel = context.WithCancelCause(ctx)
	return s
}

// SetPublicServerIPv4 sets the server public IPv4 address
func (s *Server) SetPublicServerIPv4(publicServerIP string) error {
	ip, err := netip.ParseAddr(publicServerIP)
	if err != nil {
		return fmt.Errorf("error parsing PublicServerIPv4: %w", err)
	}

	if !ip.Is4() {
		return fmt.Errorf("PublicServerIPv4 must be an IPv4 address got: %v", publicServerIP)
	}
	s.PublicServerIPv4 = ip.As4()
	return nil
}

// Listen starts the FTP Listen
func (s *Server) Listen() (err error) {

	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("error starting server: %w", err)
	}
	// Accept connections in a new goroutine

	go func() {
		<-s.ctx.Done()
		s.listener.Close()
		s.Closer <- s.ctx.Err()
	}()

	return nil
}

// Serve starts the FTP server
func (s *Server) Serve() {

	if s.TLS != nil {
		s.Logger().Debug("FTPS serve started", "addr", s.Addr)
	} else if s.TLSe != nil {
		s.Logger().Debug("FTPES serve started", "addr", s.Addr)
	} else {
		s.Logger().Debug("FTP serve started", "addr", s.Addr)
	}

	for {

		conn, err := s.listener.Accept()
		if err != nil {
			if s.ctx.Err() != nil {
				s.Logger().Info("Listener closed.")
				return
			}
			s.Logger().Error("Error accepting connection", "error", err)

			continue
		}

		if s.TLS != nil {
			s.Logger().Debug("Upgrading to TLS")
			conn, err = s.upgradeToTLS(conn, s.TLS)
			if err != nil {
				s.Logger().Error("Error upgrading to TLS", "error", err)
				return
			}
		}
		go s.ftpHandler(conn)
	}
}

// upgradeToTLS upgrades the connection to a TLS session
func (s *Server) upgradeToTLS(c net.Conn, config *tls.Config) (net.Conn, error) {
	tlsConn := tls.Server(c, config)
	if err := tlsConn.Handshake(); err != nil {
		err = fmt.Errorf("TLS Handshake error: %w", err)
		return c, err
	}
	c = tlsConn
	return c, nil
}

// ServeTLS starts the FTP server with TLS
func (s *Server) ServeTLS(certFile, keyFile string) (err error) {

	s.TLS = &tls.Config{Certificates: make([]tls.Certificate, 1)}

	s.TLS.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("error loading certificate: %w", err)
	}
	s.Serve()
	return nil
}

// ServeTLSe starts the FTP server and allow upgrade to TLS
func (s *Server) ServeTLSe(certFile, keyFile string) (err error) {

	s.TLSe = &tls.Config{Certificates: make([]tls.Certificate, 1)}

	s.TLSe.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("error loading certificate: %w", err)
	}
	s.Serve()
	return nil
}

// ListenAndServe starts the FTP server
func (s *Server) ListenAndServe() (err error) {
	err = s.Listen()
	if err != nil {
		return err
	}
	s.Serve()
	return nil
}

// ListenAndServeTLSe and allow upgrade to TLS
func (s *Server) ListenAndServeTLSe(certFile, keyFile string) (err error) {
	err = s.Listen()
	if err != nil {
		return err
	}
	err = s.ServeTLSe(certFile, keyFile)

	return
}

// ListenAndServeTLS starts the FTP server
func (s *Server) ListenAndServeTLS(certFile, keyFile string) (err error) {
	err = s.Listen()
	if err != nil {
		return err
	}
	err = s.ServeTLS(certFile, keyFile)

	return
}

// TryListenAndServe strives to starts the FTP server if there isn't an error after a certain time it returns nil
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

// TryListenAndServeTLSe strives to starts the FTP server if there isn't an error after a certain time it returns nil
func (s *Server) TryListenAndServeTLSe(certFile, keyFile string, d time.Duration) (err error) {
	errC := make(chan error)

	go func() {
		err = s.ListenAndServeTLSe(certFile, keyFile)
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

// TryListenAndServeTLS tries to start the FTP server if there isn't an error after a certain time it returns nil
func (s *Server) TryListenAndServeTLS(certFile, keyFile string, d time.Duration) (err error) {
	errC := make(chan error)

	go func() {
		err = s.ListenAndServeTLS(certFile, keyFile)
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

// Close stops the FTP server
func (s *Server) Close(err error) {
	s.cancel(err)
}

func (s *Server) Wait() error {
	return <-s.Closer
}
func (s *Server) SetLogger(l *slog.Logger) {
	s.logger = l
}
func (s *Server) Logger() *slog.Logger {
	if s.logger == nil {
		s.logger = slog.Default()
	}
	return s.logger.With("module", "ftp-server")
}
