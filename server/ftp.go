package server

import (
	"bufio"
	"fmt"
	"net"
	"runtime"
	"strings"
)

type FTPServer struct {
	listener    net.Listener
	addr        string
	supportsTLS bool
}

func NewFTPServer(addr string, fs FtpFS) *FTPServer {
	return &FTPServer{addr: addr}
}

func (f *FTPServer) Start() error {
	var err error
	// Listen on TCP port 21

	f.listener, err = net.Listen("tcp", f.addr)
	if err != nil {
		return fmt.Errorf("error starting server: %w", err)
	}
	// Accept connections in a new goroutine
	fmt.Printf("starting listener on %#+v\n", f.listener)
	go f.Run()
	return nil
}
func (f *FTPServer) Stop() error {
	fmt.Println("Stopping FTP server...")
	if f.listener == nil {
		return nil
	}
	return f.listener.Close()
}
func (f *FTPServer) Run() {
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go f.handleConnection(conn)
	}
}

func (f *FTPServer) handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	// Send a welcome message
	fmt.Fprintln(conn, "220 Welcome to My FTP Server")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			break
		}
		fmt.Println("Received:", line)
		command := strings.SplitN(strings.TrimSpace(line), " ", 2)
		fmt.Println("Received command:", command)
		cmd := command[0]
		arg := ""
		if len(command) > 1 {
			arg = command[1]
		}
		// Handle commands
		switch cmd {
		case "USER":
			fmt.Fprintln(conn, "331 Please specify the password.")
		case "PASS":
			fmt.Fprintln(conn, "230 Login successful.")
		// Add more cases here for other commands
		case "SYST":
			fmt.Fprintln(conn, "215", SendSystemType())
		case "FEAT":
			fmt.Fprintln(conn, "211-Features:")
			fmt.Fprintln(conn, " UTF8")
			fmt.Fprintln(conn, " MLST type*;size*;modify*;")
			fmt.Fprintln(conn, " MLSD")
			fmt.Fprintln(conn, " SIZE")
			fmt.Fprintln(conn, " MDTM")
			fmt.Fprintln(conn, " REST STREAM")
			//fmt.Fprintln(conn, " TVFS")
			//fmt.Fprintln(conn, " EPSV")
			//fmt.Fprintln(conn, " EPRT")
			if f.supportsTLS {
				fmt.Fprintln(conn, " AUTH TLS")
				fmt.Fprintln(conn, " AUTH SSL")
				fmt.Fprintln(conn, " PBSZ")
				fmt.Fprintln(conn, " PROT")
			}
			fmt.Fprintln(conn, "211 End")
		case "PWD":

			currentDir := "/" // Assuming root directory
			response := fmt.Sprintf("257 \"%s\" is the current directory.", currentDir)
			fmt.Fprintln(conn, response)
		case "REST":
			if arg == "0" {
				fmt.Fprintln(conn, "350 Ready for file transfer.")
			} else {
				fmt.Fprintln(conn, "350 Restarting at "+arg+". Send STORE or RETRIEVE.")
			}
		default:
			fmt.Fprintln(conn, "500 Unknown command.")
		}
	}
	conn.Close()
}

func SendSystemType() string {
	// Use runtime.GOOS to get the operating system name
	os := runtime.GOOS

	// Customize the response based on the operating system
	switch os {
	case "windows":
		return "WINDOWS Type: L8"
	case "linux", "darwin":
		return "UNIX Type: L8" // macOS is Unix-based
	default:
		return "OS Type: " + os
	}
}
