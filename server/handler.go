package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func handleConnection(conn net.Conn, manager *FTPSessionManager) {
	// Generate a unique session ID for the connection
	sessionID := generateSessionID(conn)
	session := &FTPSession{
		conn:       conn,
		workingDir: "/", // Set the initial working directory
	}

	// Add the session to the manager
	manager.Add(sessionID, session)

	// Example: Authenticate the user
	authenticateUser(session)

	// Remove the session when the client disconnects
	defer manager.Remove(sessionID)

	// Handle client commands
}

func authenticateUser(session *FTPSession) {
	// Placeholder: Implement authentication logic
	session.isAuthenticated = true // Example outcome
}

func generateSessionID(conn net.Conn) string {
	// Placeholder: Generate a unique ID for the session
	return conn.RemoteAddr().String()
}

type LogWriter struct {
	io.Writer
}

func (w *LogWriter) Write(b []byte) (int, error) {
	fmt.Println("Responding:", string(b))
	return w.Writer.Write(b)
}
func (s *FTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	logWriter := &LogWriter{conn}
	sessionID := generateSessionID(conn)
	session := &FTPSession{
		conn:            conn,
		writer:          logWriter,
		userInfo:        nil,
		workingDir:      s.root, // Set the initial working directory
		isAuthenticated: false,
		root:            s.root,
		dataListener:    nil,
		ftpServer:       s,
	}
	ftpSession := session
	// Add the session to the manager
	s.sessionManager.Add(sessionID, session)

	// Example: Authenticate the user
	authenticateUser(session)

	// Remove the session when the client disconnects
	defer s.sessionManager.Remove(sessionID)

	reader := bufio.NewReader(conn)
	// Send a welcome message
	fmt.Fprintf(conn, "220 %s\r\n", s.WelcomeMessage)

	for {

		cmd, arg, err := s.ParseCommand(reader)
		if err != nil {
			fmt.Fprintf(logWriter, err.Error())
			return
		}
		// Handle commands
		switch cmd {
		case "AUTH":
			ftpSession.AuthCommand(arg)

		// USER command is used to specify the username
		case "USER":
			err := ftpSession.UserCommand(arg)
			if err != nil {
				return
			}
		// PASS command is used to specify the password,
		// is only valid after a USER command
		case "PASS":
			err := ftpSession.PassCommand(arg)
			if err != nil {
				return
			}

		case "SYST":
			ftpSession.SystemCommand()
		case "FEAT":
			ftpSession.FeaturesCommand()
		case "OPTS":
			ftpSession.OptsCommand(arg)
		case "PWD":
			ftpSession.PrintWorkingDirectoryCommand()
		case "CWD":
			ftpSession.ChangeDirectoryCommand(arg)
		case "REST":
			ftpSession.RessetCommand(arg)
		case "TYPE":
			ftpSession.TypeCommand(arg)
		case "PASV":
			ftpSession.PasvCommand(arg)
		case "EPSV":
			ftpSession.EpsvCommand(arg)
		case "LIST":
		case "MLSD": // MLSD is LIST with machine-readable format like $ls -l
			ftpSession.MLSDCommand(arg)
		case "MLST":
			ftpSession.MLSTCommand(arg)
		case "SIZE":
			fmt.Fprintf(logWriter, "500 Unknown command.")
		case "STOR":
			ftpSession.StorCommand(arg)
		case "MDTM":
			ftpSession.ModifyTimeCommand(arg)
		case "RETR":
			ftpSession.RetrieveCommand(arg)
		case "DELE":
			ftpSession.RemoveCommand(arg)
		case "RNFR":
			ftpSession.RenameFromCommand(arg)
		case "QUIT":
			fmt.Fprintf(logWriter, "221 Goodbye.")
			return
		default:
			fmt.Fprintf(logWriter, "500 Unknown command.")
		}
	}

}

// ParseCommand  parses the command from the client and returns the command and argument.
func (s *FTPServer) ParseCommand(r *bufio.Reader) (cmd, arg string, err error) {
	line, err := r.ReadString('\n')
	if err != nil {
		err = fmt.Errorf("error reading from connection: %w", err)
		return
	}
	fmt.Println("Received:", line)
	command := strings.SplitN(strings.TrimSpace(line), " ", 2)
	cmd = command[0]

	if len(command) > 1 {
		arg = command[1]
	}
	return
}

func (s *FTPSession) AuthCommand(arg string) {
	if arg == "TLS" {
		if s.ftpServer.supportsTLS {
			fmt.Fprintf(s.writer, "234 AUTH command ok. Expecting TLS Negotiation.\r\n")
		} else {
			fmt.Fprintf(s.writer, "500 TLS not supported\r\n")
		}
	} else {
		fmt.Fprintf(s.writer, "504 AUTH command not implemented for this type\r\n")
	}
}

// UserCommand handles the USER command from the client.
func (s *FTPSession) UserCommand(arg string) (err error) {
	if arg == "" {
		err = fmt.Errorf("530 Error: User name not specified")
		fmt.Fprintf(s.writer, "%s\r\n", err.Error())
		return err
	}
	user, err := s.ftpServer.users.Get(arg)
	if err != nil {
		err = fmt.Errorf("530 Error: Searching for user failed")
		fmt.Fprintf(s.writer, "%s\r\n", err.Error())
		return
	}
	s.userInfo = user
	fmt.Fprintf(s.writer, "331 Please specify the password\r\n")
	return
}

// PassCommand handles the PASS command from the client.
func (s *FTPSession) PassCommand(arg string) (err error) {
	if s.userInfo == nil {
		err = fmt.Errorf("503 Error: User not specified")
		fmt.Fprintf(s.writer, "%s\r\n", err.Error())
		return
	}
	if s.userInfo.Password != arg {
		err = fmt.Errorf("430 Invalid username or password")
		fmt.Fprintf(s.writer, "%s\r\n", err.Error())
		return
	}
	fmt.Fprintf(s.writer, "230 Login successful\r\n")
	return
}

// SystemCommand returns the system type.
func (s *FTPSession) SystemCommand() {
	// Use runtime.GOOS to get the operating system name
	os := runtime.GOOS

	// Customize the response based on the operating system
	switch os {
	case "windows":
		fmt.Fprintf(s.writer, "215 WINDOWS Type: L8\r\n")
	case "linux", "darwin": // macOS is Unix-based
		fmt.Fprintf(s.writer, "215 UNIX Type: L8\r\n")

	default:
		fmt.Fprintf(s.writer, "215 OS Type: %s\r\n", os)
	}
}

func (s *FTPSession) FeaturesCommand() {
	fmt.Fprintf(s.writer, "211-Features:\r\n")
	fmt.Fprintf(s.writer, " UTF8\r\n")
	fmt.Fprintf(s.writer, " MLST type*;size*;modify*;\r\n")
	fmt.Fprintf(s.writer, " MLSD\r\n")
	fmt.Fprintf(s.writer, " SIZE\r\n")
	fmt.Fprintf(s.writer, " MDTM\r\n")
	fmt.Fprintf(s.writer, " REST STREAM\r\n")
	//fmt.Fprintf(s.writer, " TVFS\r\n")
	fmt.Fprintf(s.writer, " EPSV\r\n")
	//fmt.Fprintf(s.writer, " EPRT\r\n")
	if s.ftpServer.supportsTLS {
		fmt.Fprintf(s.writer, " AUTH TLS\r\n")
		fmt.Fprintf(s.writer, " AUTH SSL\r\n")
		fmt.Fprintf(s.writer, " PBSZ\r\n")
		fmt.Fprintf(s.writer, " PROT\r\n")
	}
	fmt.Fprintf(s.writer, "211 End\r\n")

}

// PrintWorkingDirectoryCommand handles the PWD command from the client.
// The PWD command is used to print the current working directory on the server.
func (s *FTPSession) PrintWorkingDirectoryCommand() {
	fmt.Fprintf(s.writer, "257 \"%s\" is current directory\r\n", s.workingDir)
}

// ChangeDirectoryCommand handles the CWD command from the client.
// The CWD command is used to change the working directory on the server.
func (s *FTPSession) ChangeDirectoryCommand(arg string) {

	requestedDir := Abs(arg, s.root, s.workingDir)
	fmt.Println("requestedDir:", requestedDir)
	err := s.ftpServer.fs.CheckDir(requestedDir)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error: %s\r\n", err.Error())
	}

	s.workingDir = requestedDir
	fmt.Fprintf(s.writer, "250 Directory successfully changed to \"%s\"\r\n", arg)

}

func Abs(arg string, root string, workingDir string) string {
	if len(arg) == 0 {
		return "."
	}
	fmt.Println("arg:", arg, "root:", root, "workingDir:", workingDir, strings.HasPrefix(arg, root))
	if strings.HasPrefix(arg, root) {
		return arg
	}
	return filepath.Join(workingDir, arg)

}
func (s *FTPSession) RessetCommand(arg string) {
	if arg == "0" {
		fmt.Fprintf(s.writer, "350 Ready for file transfer.\r\n")
	} else {
		fmt.Fprintf(s.writer, "350 Restarting at "+arg+". Send STORE or RETRIEVE.\r\n")
	}
}

// OptsCommand handles the OPTS command from the client.
// The OPTS command is used to specify options for the server.
func (s *FTPSession) OptsCommand(arg string) {
	switch arg {
	case "UTF8 ON":
		fmt.Fprintf(s.writer, "200 Always in UTF8 mode.\r\n")

	default:
		fmt.Fprintf(s.writer, "500 Unknown option.\r\n")
	}
}

// TypeCommand handles the TYPE command from the client.
// The TYPE command is used to specify the type of file being transferred.
// The two types are ASCII (A) and binary (I).
func (s *FTPSession) TypeCommand(arg string) {
	if arg == "I" {
		s.ftpServer.Type = typeI
		fmt.Fprintf(s.writer, "200 Type set to I\r\n")
	} else if arg == "A" {
		s.ftpServer.Type = typeA
		fmt.Fprintf(s.writer, "200 Type set to A\r\n")
	} else {
		fmt.Fprintf(s.writer, "500 Unknown type\r\n")
	}
}

// findAvailablePortInRange finds an available port in the given range.
// It returns a listener on the available port and the port number.
func findAvailablePortInRange(start, end int) (net.Listener, int, error) {
	for port := start; port <= end; port++ {
		address := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			return listener, port, nil
		}
	}
	return nil, 0, fmt.Errorf("no available ports found in range %d-%d", start, end)
}

// PasvEpsvCommand handles the PASV command from the client.
// The PASV command is used to enter passive mode.
func (s *FTPSession) PasvEpsvCommand(arg string) (port int, err error) {

	dataListener, port, err := findAvailablePortInRange(s.ftpServer.pasvMinPort, s.ftpServer.pasvMaxPort)
	if err != nil {
		fmt.Fprintf(s.writer, "500: Server error listening for data connection: %s\r\n", err.Error())
		return 0, err
	}

	s.dataListener = dataListener
	// Extract the port from the listener's address
	_, portString, err := net.SplitHostPort(dataListener.Addr().String())
	if err != nil {
		fmt.Fprintf(s.writer, "500 Server error getting port: %s\r\n", err.Error())
		dataListener.Close()
	}
	port, err = strconv.Atoi(portString)
	if err != nil {
		fmt.Fprintf(s.writer, "500 Server error with port conversion: %s\r\n", err.Error())
		dataListener.Close()
	}
	return port, nil
}

// PasvCommand handles the PASV command from the client.
// The PASV command is used to enter passive mode.
func (s *FTPSession) PasvCommand(arg string) error {
	port, err := s.PasvEpsvCommand(arg)
	if err != nil {
		return err
	}
	PublicIP := s.ftpServer.PublicServerIP

	fmt.Fprintf(s.writer, "227 Entering Passive Mode (%d,%d,%d,%d,%d,%d)\r\n",
		PublicIP[0], PublicIP[1], PublicIP[2], PublicIP[3], port/256, port%256)
	return nil
}

// EpsvCommand handles the EPSV command from the client.
// The EPSV command is used to enter extended passive mode.
func (s *FTPSession) EpsvCommand(arg string) error {
	// Listen on a new port
	port, err := s.PasvEpsvCommand(arg)
	if err != nil {
		return err
	}

	// Respond with the port number
	// The response format is 229 Entering Extended Passive Mode (|||port|)
	fmt.Fprintf(s.writer, "229 Entering Extended Passive Mode (|||%d|)\r\n", port)
	return nil

}

// StorCommand handles the STOR command from the client.
// The STOR command is used to store a file on the server.
func (s *FTPSession) StorCommand(arg string) {
	// Close the data connection
	defer s.dataListener.Close()
	// At this point, dataConn is ready for use for data transfer
	// You can now send or receive data over dataConn
	fmt.Fprintf(s.writer, "150 Opening data connection.\r\n")
	// Wait for the client to connect on this new port
	dataConn, err := s.dataListener.Accept()
	if err != nil {
		fmt.Fprintf(s.writer, "425 Can't open data connection: %s\r\n", err)
		return
	}
	defer dataConn.Close()
	filename := filepath.Join(s.workingDir, arg)
	err = s.ftpServer.fs.Create(filename, dataConn, string(s.ftpServer.Type))
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error writing to the file: %s\r\n", err.Error())
		return
	}
	fmt.Fprintf(s.writer, "226 Transfer complete\r\n")

}

// ModifyTimeCommand handles the MDTM command from the client.
// The MDTM command is used to modify the modification time of a file on the server.
func (s *FTPSession) ModifyTimeCommand(arg string) {
	args := strings.SplitN(arg, " ", 2)
	if len(args) == 0 {
		fmt.Fprintf(s.writer, "501 No file name given\r\n")
		return
	} else if len(args) == 1 {
		stat, err := s.ftpServer.fs.Stat(args[0])
		if err != nil {
			fmt.Fprintf(s.writer, "501 Error getting file info: %s\r\n", err)
			return
		}
		fmt.Fprintf(s.writer, "213 %s\r\n", stat)
	} else if len(args) == 2 {
		err := s.ftpServer.fs.ModifyTime(args[1], args[0])
		if err != nil {
			fmt.Fprintf(s.writer, "501 Error setting file '%s' time '%s' modification time: %s\r\n", args[1], args[0], err.Error())
			return
		}
		fmt.Fprintf(s.writer, "213 File modification time set to: %s\r\n", args[0])
	}

}
func (s *FTPSession) CloseDataConnection() {
	// Close the data connection
	if s.dataListener != nil {
		s.dataListener.Close()
	}
}

// MLSDCommand handles the MLSD command from the client.
// The MLSD command is used to list the contents of a directory in a machine-readable format.
func (s *FTPSession) MLSDCommand(arg string) {
	// Close the data connection

	fmt.Fprintf(s.writer, "150 Here comes the directory listing.\r\n")
	dataConn, err := s.dataListener.Accept()

	if err != nil {
		fmt.Fprintf(s.writer, "425 Can't open data connection: %s\r\n", err.Error())
	}

	// Send the directory listing
	// Send the directory listing
	entries, err := s.ftpServer.fs.Dir(s.workingDir)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error getting directory listing. error: %s\r\n", err.Error())
		return
	}

	for _, entry := range entries {
		fmt.Println("dataConn:", entry)
		fmt.Fprintf(dataConn, "%s\r\n", entry)
	}
	dataConn.Close()
	s.dataListener.Close()
	fmt.Fprintf(s.writer, "226 Directory send OK.\r\n")
}
func (s *FTPSession) MLSTCommand(arg string) {
	filename := ""
	if len(arg) > 0 {
		if strings.HasPrefix(arg, s.root) {
			filename = arg
		} else {
			filename = filepath.Join(s.workingDir, arg)
		}

	}

	entries, err := s.ftpServer.fs.Stat(filename)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error getting file info: %s\r\n", err.Error())
		return
	}
	fmt.Fprintf(s.writer, "250-File details:\r\n")
	fmt.Fprintf(s.writer, " %s\r\n", entries)
	fmt.Fprintf(s.writer, "250 End\r\n")
}
func (s *FTPSession) RetrieveCommand(arg string) {

	// Close the data connection
	defer s.dataListener.Close()
	// At this point, dataConn is ready for use for data transfer
	// You can now send or receive data over dataConn
	fmt.Fprintf(s.writer, "150 Opening data connection.\r\n")
	// Wait for the client to connect on this new port
	dataConn, err := s.dataListener.Accept()
	if err != nil {
		fmt.Fprintf(s.writer, "425 Can't open data connection: %s\r\n", err.Error())
	}
	filename := filepath.Join(s.workingDir, arg)
	fmt.Println("RETR:", filename)
	_, err = s.ftpServer.fs.Read(filename, dataConn)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error reading the file: %s\r\n", err.Error())
	}
	dataConn.Close()
	s.dataListener.Close()
	fmt.Fprintf(s.writer, "226 Transfer complete\r\n")
}

func (s *FTPSession) RemoveCommand(arg string) {
	fileName := filepath.Join(s.workingDir, arg)
	err := s.ftpServer.fs.Remove(fileName)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error deleting file: %s\r\n", err.Error())
		return
	}
	fmt.Fprintf(s.writer, "250 File deleted")
}
func (s *FTPSession) RenameFromCommand(arg string) {

}
