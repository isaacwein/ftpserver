package server

import (
	"fmt"
	"github.com/telebroad/ftpserver/users"
	"net"
	"net/netip"
)

type FTPServerTransferType string

const (
	typeA FTPServerTransferType = "A"
	typeI FTPServerTransferType = "I"
)

type FTPServer struct {
	listener       net.Listener
	addr           string
	supportsTLS    bool
	fs             FtpFS
	root           string
	sessionManager *FTPSessionManager
	users          users.Users
	WelcomeMessage string
	PublicServerIP [4]byte
	Type           FTPServerTransferType
	pasvMaxPort    int
	pasvMinPort    int
}

func NewFTPServer(addr, PublicServerIP string, fs FtpFS, users users.Users, pasvMinPort, pasvMaxPort int) (*FTPServer, error) {
	ip, err := netip.ParseAddr(PublicServerIP)
	if err != nil {
		return nil, fmt.Errorf("error parsing PublicServerIP: %w", err)
	}
	if !ip.Is4() {
		return nil, fmt.Errorf("PublicServerIP must be an IPv4 address got: %v", PublicServerIP)
	}

	return &FTPServer{
		addr:           addr,
		fs:             fs,
		sessionManager: NewSessionManager(),
		users:          users,
		root:           fs.RootDir(),
		WelcomeMessage: "Welcome to My FTP Server",
		PublicServerIP: ip.As4(),
		pasvMaxPort:    pasvMaxPort,
		pasvMinPort:    pasvMinPort,
	}, nil
}

func (s *FTPServer) Start() error {
	var err error
	// Listen on TCP port 21

	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("error starting server: %w", err)
	}
	// Accept connections in a new goroutine
	fmt.Printf("starting listener on %#+v\n", s.addr)
	go s.Run()
	return nil
}

func (s *FTPServer) Stop() error {
	fmt.Println("Stopping FTP server...")
	if s.listener == nil {
		return nil
	}
	return s.listener.Close()
}
func (s *FTPServer) Run() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go s.handleConnection(conn)
	}
}
