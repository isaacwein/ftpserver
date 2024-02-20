package ftp

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Server struct {
	listener net.Listener

	supportsTLS bool

	// Addr optionally specifies the TCP address for the server to listen on,
	// in the form "host:port". If empty, ":http" (port 80) is used.
	// The service names are defined in RFC 6335 and assigned by IANA.
	// See net.Dial for details of the address format.
	Addr string

	// handler to invoke, http.DefaultServeMux if nil
	Handler Handler

	// TLSConfig optionally provides a TLS configuration for use
	// by ServeTLS and ListenAndServeTLS. Note that this value is
	// cloned by ServeTLS and ListenAndServeTLS, so it's not
	// possible to modify the configuration with methods like
	// tls.Config.SetSessionTicketKeys. To use
	// SetSessionTicketKeys, use Server.Serve with a TLS Listener
	// instead.
	TLSConfig *tls.Config
	// to upgrade a non TLS to a TLS session
	TLSeConfig *tls.Config
	// to upgrade a non TLS to a SSH session
	SSHConfig *tls.Config

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body. A zero or negative value means
	// there will be no timeout.
	//
	// Because ReadTimeout does not let Handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	// A zero or negative value means there will be no timeout.
	WriteTimeout time.Duration

	// ErrorLog specifies an optional logger for errors accepting
	// connections, unexpected behavior from handlers, and
	// underlying FileSystem errors.
	// If nil, logging is done via the log package's standard logger.
	ErrorLog *log.Logger

	// BaseContext optionally specifies a function that returns
	// the base context for incoming requests on this server.
	// The provided Listener is the specific Listener that's
	// about to start accepting requests.
	// If BaseContext is nil, the default is context.Background().
	// If non-nil, it must return a non-nil context.
	BaseContext func(net.Listener) context.Context

	// ConnContext optionally specifies a function that modifies
	// the context used for a new connection c. The provided ctx
	// is derived from the base context and has a ServerContextKey
	// value.
	ConnContext func(ctx context.Context, c net.Conn) context.Context

	mu         sync.Mutex
	listeners  map[*net.Listener]struct{}
	activeConn map[*Session]struct{} // Map of active sessions

	listenerGroup sync.WaitGroup // Protects the sessions map
	onShutdown    []func()
}

func (s *Server) Serve() {
	if s.Handler == nil {
		s.Handler = DefaultServeMux
	}
	for {
		conn, err := s.listener.Accept()

		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) ListenAndServe() error {

	fmt.Println("Starting FTP server...")

	var err error
	// Listen on TCP port 21

	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("error starting server: %w", err)
	}
	// Accept connections in a new goroutine
	fmt.Printf("starting listener on %#+v\n", s.listener)

	if s.TLSConfig != nil {
		s.supportsTLS = true
	}

	go s.Serve()
	return nil
}

func (s *Server) ListenAndServeTLS(certFile, keyFile fs.File) (err error) {

	cert, err := io.ReadAll(certFile)
	if err != nil {
		return fmt.Errorf("error reading certificate: %w", err)
	}
	key, err := io.ReadAll(keyFile)
	if err != nil {
		return fmt.Errorf("error reading key: %w", err)
	}

	config := &tls.Config{}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.X509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("error loading certificate: %w", err)
	}
	s.TLSConfig = config
	return s.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	for _, f := range s.onShutdown {
		go f()
	}
	s.mu.Unlock()
	return nil
}

func ListenAndServe() error {
	server := &Server{}
	return server.ListenAndServe()
}

type Request struct {
	Method Command
	Proto  string // e.g. "FTP","FTPS","SFTP","FTPES
	Body   []byte
	// RemoteAddr allows HTTP servers and other software to record
	// the network address that sent the request, usually for
	// logging. This field is not filled in by ReadRequest and
	// has no defined format. The HTTP server in this package
	// sets RemoteAddr to an "IP:port" address before invoking a
	// handler.
	// This field is ignored by the HTTP client.
	RemoteAddr net.Addr

	// RequestURI is the unmodified request-target of the
	// Request-Line (RFC 7230, Section 3.1.1) as sent by the client
	// to a server. Usually the URL field should be used instead.
	// It is an error to set this field in an HTTP client request.
	RequestURI net.Addr

	// TLS allows HTTP servers and other software to record
	// information about the TLS connection on which the request
	// was received. This field is not filled in by ReadRequest.
	// The HTTP server in this package sets the field for
	// TLS-enabled connections before invoking a handler;
	// otherwise it leaves the field nil.
	// This field is ignored by the HTTP client.

	TLS  *tls.ConnectionState // TLS for FTPS
	TLSe *tls.ConnectionState // TLS for FTPES
	SSH  *tls.ConnectionState // TLS for SFTP
}

type Response struct {
	Status     string // e.g. "200 OK"
	StatusCode int    // e.g. 200

	// Close records whether the header directed that the connection be
	// closed after reading Body. The value is advice for clients: neither
	// ReadResponse nor Response.Write ever closes a connection.
	Close bool

	// TLS contains information about the TLS connection on which the
	// response was received. It is nil for unencrypted responses.
	// The pointer is shared between responses and should not be
	// modified.
	TLS *tls.ConnectionState
	// TLSe contains information about the TLS connection on which the
	TLSe *tls.ConnectionState
	// SSH response was received. It is nil for unencrypted responses.
	SSH *tls.ConnectionState

	conn net.Conn
}

func (r *Response) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.Status = StatusText(statusCode)
}

func (r *Response) Write(p []byte) (n int, err error) {
	return r.conn.Write(p)
}

type ResponseWriter interface {
	WriteHeader(statusCode int)
	Write(p []byte) (n int, err error)
}
type Handler interface {
	ServeFTP(w ResponseWriter, r *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

// ServeFTP calls f(w, r).
func (f HandlerFunc) ServeFTP(w ResponseWriter, r *Request) {
	f(w, r)
}
func (s *Server) handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)

	peek, err := reader.Peek(4)
	if err != nil {
		fmt.Println("Error reading from connection:", err)
		return
	}
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {

			return
		}

		request := &Request{
			Method:     strings.Trim(string(peek), ""),
			Proto:      "",
			Body:       line,
			RemoteAddr: conn.RemoteAddr(),
			RequestURI: conn.LocalAddr(),
			TLS:        nil,
			SSH:        nil,
		}
		response := &Response{}
		s.Handler.ServeFTP(response, request)

	}

}
