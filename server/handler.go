package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

func handleConnection(conn net.Conn, manager *SessionManager) {
	// Generate a unique session ID for the connection
	sessionID := generateSessionID(conn)
	session := &Session{
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

func authenticateUser(session *Session) {
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

type handlerMap map[string]func(cmd string, arg string) error

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "Recovered from panic:", r)
			fmt.Fprintln(os.Stderr, "stack:", string(debug.Stack()))
		}
	}()
	defer conn.Close()
	logWriter := &LogWriter{conn}
	sessionID := generateSessionID(conn)
	session := &Session{
		conn:            conn,
		writer:          logWriter,
		userInfo:        nil,
		workingDir:      s.Root, // Set the initial working directory
		isAuthenticated: false,
		root:            s.Root,
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
	if string(s.PublicServerIPv4[:]) == "" {

		addr, err := netip.ParseAddr(conn.LocalAddr().String())
		if err != nil {
			fmt.Fprintf(logWriter, "error getting local ip: %s.\r\n", err.Error())
			fmt.Fprintf(os.Stderr, "error getting local ip: %s\n", err.Error())
			return
		}
		s.PublicServerIPv4 = addr.As4()
	}
	reader := bufio.NewReader(conn)
	// Send a welcome message
	fmt.Fprintf(conn, "220 %s\r\n", s.WelcomeMessage)
	handlers := handlerMap{
		"AUTH": ftpSession.AuthCommand,                    // AUTH is used to authenticate the client
		"USER": ftpSession.UserCommand,                    // USER is used to specify the username
		"PASS": ftpSession.PassCommand,                    // PASS is used to specify the password
		"SYST": ftpSession.SystemCommand,                  // SYST is used to get the system type
		"FEAT": ftpSession.FeaturesCommand,                // FEAT is used to get the supported features
		"OPTS": ftpSession.OptsCommand,                    // OPTS is used to specify options for the server
		"HELP": ftpSession.HelpCommand,                    // HELP is used to get help
		"NOOP": ftpSession.NoopCommand,                    // NOOP is used to keep the connection alive
		"PWD":  ftpSession.PrintWorkingDirectoryCommand,   // PWD is used to print the current working directory
		"CWD":  ftpSession.ChangeDirectoryCommand,         // CWD is used to change the working directory
		"CDUP": ftpSession.ChangeDirectoryToParentCommand, // CDUP is used to change the working directory to the parent directory
		"REST": ftpSession.RessetCommand,                  // REST is used to restart the file transfer
		"TYPE": ftpSession.TypeCommand,                    // TYPE is used to specify the type of file being transferred
		"MODE": ftpSession.ModeCommand,                    // MODE is used to specify the transfer mode (stream, block, or compressed)
		"STRU": ftpSession.StruCommand,                    // STRU is used to specify the file structure (file, record, or page)
		"PASV": ftpSession.PassiveModeCommand,             // PASV is used to enter passive mode
		"EPSV": ftpSession.ExtendedPassiveModeCommand,     // EPSV is used to enter extended passive mode
		"PORT": ftpSession.ActiveModeCommand,              // PORT is used to specify an address and port to which the server should connect
		"EPRT": ftpSession.ExtendedActiveModeCommand,      // EPRT is used to specify an address and port to which the server should connect
		"ABOR": ftpSession.AbortCommand,                   // ABOR is used to abort the previous FTP command
		"MLSD": ftpSession.GetDirInfoCommand,              // MLSD is LIST with machine-readable format like $ls -l
		"MLST": ftpSession.GetFileInfoCommand,             // MLST is used to get information about a file
		"STAT": ftpSession.GetFileInfoCommand,             // MLST is used to get information about a file
		"SIZE": ftpSession.SizeCommand,                    // SIZE is used to get the size of a file
		"STOR": ftpSession.SaveCommand,                    // STOR is used to store a file on the server
		"APPE": ftpSession.SaveCommand,                    // APPE is used to append to a file on the server
		"MDTM": ftpSession.ModifyTimeCommand,              // MDTM is used to modify the modification time of a file
		"RETR": ftpSession.RetrieveCommand,                // RETR is used to retrieve a file from the server
		"DELE": ftpSession.RemoveCommand,                  // DELE is used to delete a file
		"RNFR": ftpSession.RenameFromCommand,              // RNFR is used to specify the file to be renamed
		"RNTO": ftpSession.RenameToCommand,                // RNTO is used to specify the new name for the file
		"QUIT": ftpSession.CloseCommand,                   // QUIT is used to terminate the connection
	}
	HelpCommands := make([]string, 0, len(handlers))
	for k := range handlers {
		HelpCommands = append(HelpCommands, k)
	}
	ftpSession.HelpCommands = strings.Join(HelpCommands, " ")

	for {

		cmd, arg, err := s.ParseCommand(reader)
		if err != nil {
			fmt.Fprintf(logWriter, err.Error())
			return
		}

		if command, ok := handlers[cmd]; ok {
			err := command(cmd, arg)
			if err != nil {
				return
			}
		} else {
			ftpSession.UnknownCommand(cmd, arg)
		}
	}
}

// ParseCommand  parses the command from the client and returns the command and argument.
func (s *Server) ParseCommand(r *bufio.Reader) (cmd, arg string, err error) {
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

// AuthCommand handles the AUTH command from the client.
func (s *Session) AuthCommand(cmd, arg string) error {
	if arg == "TLS" {
		if s.ftpServer.supportsTLS {
			fmt.Fprintf(s.writer, "234 AUTH command ok. Expecting TLS Negotiation.\r\n")
		} else {
			fmt.Fprintf(s.writer, "500 TLS not supported\r\n")
		}
	} else {
		fmt.Fprintf(s.writer, "504 AUTH command not implemented for this type\r\n")
	}
	return nil
}

// UserCommand handles the USER command from the client.
func (s *Session) UserCommand(cmd, arg string) (err error) {
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
func (s *Session) PassCommand(cmd, arg string) (err error) {
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
func (s *Session) SystemCommand(cmd, arg string) error {
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
	return nil
}

func (s *Session) FeaturesCommand(cmd, arg string) error {
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
	return nil
}

// HelpCommand handles the HELP command from the client.
func (s *Session) HelpCommand(cmd, arg string) error {
	fmt.Fprintf(s.writer, "214-The following commands are recognized.\r\n")

	fmt.Fprintf(s.writer, " %s\r\n", s.HelpCommands)
	fmt.Fprintf(s.writer, "214 Help OK.\r\n")
	return nil

}

// NoopCommand handles the NOOP command from the client.
// The NOOP command is used to keep the connection alive.
func (s *Session) NoopCommand(cmd, arg string) error {
	fmt.Fprintf(s.writer, "200 NOOP ok.\r\n")
	return nil
}

// PrintWorkingDirectoryCommand handles the PWD command from the client.
// The PWD command is used to print the current working directory on the server.
func (s *Session) PrintWorkingDirectoryCommand(cmd, arg string) error {
	fmt.Fprintf(s.writer, "257 \"%s\" is current directory\r\n", s.workingDir)
	return nil
}

// ChangeDirectoryCommand handles the CWD command from the client.
// The CWD command is used to change the working directory on the server.
func (s *Session) ChangeDirectoryCommand(cmd, arg string) error {

	requestedDir := Abs(s.root, s.workingDir, arg)
	fmt.Println("requestedDir:", requestedDir)
	err := s.ftpServer.FsHandler.CheckDir(requestedDir)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error: %s\r\n", err.Error())
		return nil
	}

	s.workingDir = requestedDir
	fmt.Fprintf(s.writer, "250 Directory successfully changed to \"%s\"\r\n", requestedDir)
	return nil

}

// ChangeDirectoryToParentCommand handles the CDUP command from the client.
// The CDUP command is used to change the working directory to the parent directory.
func (s *Session) ChangeDirectoryToParentCommand(cmd, arg string) error {

	requestedDir := Abs(s.root, s.workingDir, "..")
	fmt.Println("requestedDir:", requestedDir)
	err := s.ftpServer.FsHandler.CheckDir(requestedDir)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error: %s\r\n", err.Error())
		return nil
	}

	s.workingDir = requestedDir
	fmt.Fprintf(s.writer, "250 Directory successfully changed to \"%s\"\r\n", requestedDir)
	return nil
}

func Abs(root string, workingDir string, arg string) string {
	if len(arg) == 0 {
		return "."
	}
	if strings.HasPrefix(arg, root) {
		return arg
	}
	return filepath.Join(workingDir, arg)

}
func (s *Session) RessetCommand(cmd, arg string) error {
	if arg == "0" {
		fmt.Fprintf(s.writer, "350 Ready for file transfer.\r\n")
	} else {
		fmt.Fprintf(s.writer, "350 Restarting at "+arg+". Send STORE or RETRIEVE.\r\n")
	}
	return nil
}

// OptsCommand handles the OPTS command from the client.
// The OPTS command is used to specify options for the server.
func (s *Session) OptsCommand(cmd, arg string) error {
	switch arg {
	case "UTF8 ON":
		fmt.Fprintf(s.writer, "200 Always in UTF8 mode.\r\n")

	default:
		fmt.Fprintf(s.writer, "500 Unknown option.\r\n")
	}
	return nil
}

// TypeCommand handles the TYPE command from the client.
// The TYPE command is used to specify the type of file being transferred.
// The two types are ASCII (A) and binary (I).
func (s *Session) TypeCommand(cmd, arg string) error {
	if arg == "I" {
		s.ftpServer.Type = typeI
		fmt.Fprintf(s.writer, "200 Type set to I\r\n")
	} else if arg == "A" {
		s.ftpServer.Type = typeA
		fmt.Fprintf(s.writer, "200 Type set to A\r\n")
	} else {
		fmt.Fprintf(s.writer, "500 Unknown type\r\n")
	}
	return nil
}

// ModeCommand handles the MODE command from the client.
func (s *Session) ModeCommand(cmd, args string) error {
	if args == "S" { // Stream mode
		fmt.Fprintf(s.writer, "200 Mode set to S.\r\n")
	} else {
		// Other modes are not commonly supported or required
		fmt.Fprintf(s.writer, "504 Unsupported mode.\r\n")
	}
	return nil
}

// StruCommand handles the SIZE command from the client.
func (s *Session) StruCommand(cmd, args string) error {
	if args == "F" { // File structure

		fmt.Fprintf(s.writer, "200 Structure set to F.\r\n")
	} else {
		// Other structures are not commonly supported or required
		fmt.Fprintf(s.writer, "504 Structure %s not implemented.\r\n", args)
	}
	return nil
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
func (s *Session) PasvEpsvCommand(arg string) (port int, err error) {

	dataListener, port, err := findAvailablePortInRange(s.ftpServer.PasvMinPort, s.ftpServer.PasvMaxPort)
	if err != nil {
		fmt.Fprintf(s.writer, "500: Server error listening for data connection: %s\r\n", err.Error())
		return 0, err
	}

	s.dataListener = dataListener
	// Extract the port from the listener's address
	_, portString, err := net.SplitHostPort(dataListener.Addr().String())
	if err != nil {
		fmt.Fprintf(s.writer, "500 Server error getting port: %s\r\n", err.Error())
		s.CloseDataConnection()
		return 0, err
	}
	port, err = strconv.Atoi(portString)
	if err != nil {
		fmt.Fprintf(s.writer, "500 Server error with port conversion: %s\r\n", err.Error())
		s.CloseDataConnection()
	}
	return port, err
}
func (s *Session) PortErptCommand(addr string) (err error) {
	s.dataCaller, err = net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintf(s.writer, "500 Server error connecting to data port: %s\r\n", err.Error())
	}
	return err
}

// PassiveModeCommand handles the PASV command from the client.
// The PASV command is used to enter passive mode.
func (s *Session) PassiveModeCommand(cmd, arg string) error {
	port, err := s.PasvEpsvCommand(arg)
	if err != nil {
		return nil
	}
	PublicIP := s.ftpServer.PublicServerIPv4

	fmt.Fprintf(s.writer, "227 Entering Passive Mode (%d,%d,%d,%d,%d,%d)\r\n",
		PublicIP[0], PublicIP[1], PublicIP[2], PublicIP[3], port/256, port%256)
	return nil
}

// ExtendedPassiveModeCommand handles the EPSV command from the client.
// The EPSV command is used to enter extended passive mode.
func (s *Session) ExtendedPassiveModeCommand(cmd, arg string) error {
	// Listen on a new port
	port, err := s.PasvEpsvCommand(arg)
	if err != nil {
		return nil
	}

	// Respond with the port number
	// The response format is 229 Entering Extended Passive Mode (|||port|)
	fmt.Fprintf(s.writer, "229 Entering Extended Passive Mode (|||%d|)\r\n", port)
	return nil

}

// ActiveModeCommand handles the PORT command from the client.
func (s *Session) ActiveModeCommand(cmd, args string) error {
	parts := strings.Split(args, ",")
	if len(parts) != 6 {
		fmt.Fprintf(s.writer, "501 Syntax error in parameters or arguments.")
		return nil
	}

	ip := strings.Join(parts[0:4], ".")
	p1, _ := strconv.Atoi(parts[4])
	p2, _ := strconv.Atoi(parts[5])
	port := p1*256 + p2
	err := s.PortErptCommand(fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil
	}
	// Here you would prepare to open a data connection using the parsed IP and port.
	fmt.Fprintf(s.writer, "200 PORT command successful.")
	return nil
}

// ExtendedActiveModeCommand handles the EPRT command from the client.
func (s *Session) ExtendedActiveModeCommand(cmd, arg string) error {
	parts := strings.Split(arg, "|")
	if len(parts) != 5 || (parts[1] != "1" && parts[1] != "2") { // 1 for IPv4, 2 for IPv6
		fmt.Fprintf(s.writer, "501 Syntax error in parameters or arguments.")
		return nil
	}

	ip := parts[2]
	port := parts[3]
	err := s.PortErptCommand(fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil
	}

	// Here you would prepare to open a data connection using the parsed IP and port.
	fmt.Fprintf(s.writer, "200 EPRT command successful.")

	return nil
}

// PassiveOrActiveModeConn returns the data connection.
// if passive mode is enabled, it returns the listener.
// if active mode is enabled, it returns the caller.
func (s *Session) PassiveOrActiveModeConn() (net.Conn, error) {
	if s.dataListener != nil {
		return s.dataListener.Accept()
	}
	if s.dataCaller != nil {
		return s.dataCaller, nil
	}
	return nil, fmt.Errorf("no data connection")
}

// AbortCommand handles the ABOR command from the client.
func (s *Session) AbortCommand(cmd, arg string) error {
	if s.dataListener != nil {
		s.CloseDataConnection()
	}
	if s.dataCaller != nil {
		s.CloseDataCaller()
	}

	fmt.Fprintf(s.writer, "226 ABOR command successful.\r\n")
	return nil
}

// CloseDataConnection closes the data connection.
func (s *Session) CloseDataConnection() {
	// Close the data connection
	if s.dataListener != nil {
		s.dataListener.Close()
		s.dataListener = nil
	}
}

// CloseDataCaller closes the data connection.
func (s *Session) CloseDataCaller() {
	// Close the data connection
	if s.dataCaller != nil {
		s.dataCaller.Close()
		s.dataCaller = nil
	}
}

// SaveCommand handles the STOR command from the client.
// The STOR command is used to store a file on the server.
func (s *Session) SaveCommand(cmd, arg string) error {
	// Close the data connection
	defer s.CloseDataConnection()
	// At this point, dataConn is ready for use for data transfer
	// You can now send or receive data over dataConn
	fmt.Fprintf(s.writer, "150 Opening data connection.\r\n")
	// Wait for the client to connect on this new port

	dataConn, err := s.PassiveOrActiveModeConn()
	if err != nil {
		fmt.Fprintf(s.writer, "425 Can't open data connection: %s\r\n", err)
		return nil
	}
	defer dataConn.Close()
	filename := Abs(s.root, s.workingDir, arg)
	appendOnly := false
	if cmd == "APPE" {
		appendOnly = true
	}

	err = s.ftpServer.FsHandler.Create(filename, dataConn, string(s.ftpServer.Type), appendOnly)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error writing to the file: %s\r\n", err.Error())
		return nil

	}
	fmt.Fprintf(s.writer, "226 Transfer complete\r\n")
	return nil
}

// ModifyTimeCommand handles the MDTM command from the client.
// The MDTM command is used to modify the modification time of a file on the server.
func (s *Session) ModifyTimeCommand(cmd, arg string) error {
	args := strings.SplitN(arg, " ", 2)
	if len(args) == 0 {
		fmt.Fprintf(s.writer, "501 No file name given\r\n")
		return nil
	} else if len(args) == 1 {
		stat, _, err := s.ftpServer.FsHandler.Stat(args[0])
		if err != nil {
			fmt.Fprintf(s.writer, "501 Error getting file info: %s\r\n", err)
			return nil
		}
		fmt.Fprintf(s.writer, "213 %s\r\n", stat)
	} else if len(args) == 2 {
		err := s.ftpServer.FsHandler.ModifyTime(args[1], args[0])
		if err != nil {
			fmt.Fprintf(s.writer, "501 Error setting file '%s' time '%s' modification time: %s\r\n", args[1], args[0], err.Error())
			return nil
		}
		fmt.Fprintf(s.writer, "213 File modification time set to: %s\r\n", args[0])
	}
	return nil
}

// GetDirInfoCommand handles the MLSD command from the client.
// The MLSD command is used to list the contents of a directory in a machine-readable format.
func (s *Session) GetDirInfoCommand(cmd, arg string) error {
	// Close the data connection
	defer s.CloseDataConnection()
	fmt.Fprintf(s.writer, "150 Here comes the directory listing.\r\n")
	dataConn, err := s.PassiveOrActiveModeConn()

	if err != nil {
		fmt.Fprintf(s.writer, "425 Can't open data connection: %s\r\n", err.Error())
		return nil
	}
	defer dataConn.Close()
	// Send the directory listing
	// Send the directory listing
	entries, err := s.ftpServer.FsHandler.Dir(s.workingDir)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error getting directory listing. error: %s\r\n", err.Error())
		return nil
	}

	for _, entry := range entries {
		fmt.Println("dataConn:", entry)
		fmt.Fprintf(dataConn, "%s\r\n", entry)
	}

	fmt.Fprintf(s.writer, "226 Directory send OK.\r\n")
	return nil
}

// StatusCommand handles the MLST command from the client.
func (s *Session) StatusCommand(cmd, arg string) error {

	if arg == "" {
		fmt.Fprintf(s.writer, "211-FTP Server Status:\n")
		fmt.Fprintf(s.writer, "211 End of status.\r\n")
		return nil
	} else {

		fmt.Fprintf(s.writer, "213-Status of %s:\n", arg)
		filename := Abs(s.root, s.workingDir, arg)

		entries, _, err := s.ftpServer.FsHandler.Stat(filename)
		if err != nil {
			fmt.Fprintf(s.writer, "550 Error getting file info: %s\n", err.Error())
			return nil
		}
		fmt.Fprintf(s.writer, " %s\n", entries)
		fmt.Fprintf(s.writer, "213 End of status.\r\n")
	}

	return nil
}

// GetFileInfoCommand handles the MLST command from the client.
func (s *Session) GetFileInfoCommand(cmd, arg string) error {
	filename := Abs(s.root, s.workingDir, arg)

	entries, _, err := s.ftpServer.FsHandler.Stat(filename)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error getting file info: %s\r\n", err.Error())
		return nil
	}
	fmt.Fprintf(s.writer, "250-File details:\n")
	fmt.Fprintf(s.writer, " %s\n", entries)
	fmt.Fprintf(s.writer, "250 End\r\n")
	return nil
}

// SizeCommand handles the SIZE command from the client.
func (s *Session) SizeCommand(cmd, arg string) error {
	filename := Abs(s.root, s.workingDir, arg)

	_, fileInfo, err := s.ftpServer.FsHandler.Stat(filename)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error getting file info: %s\r\n", err.Error())
		return nil
	}
	// File exists; return its size
	fmt.Fprintf(s.writer, "213 %d\r\n", fileInfo.Size())
	return nil
}

// RetrieveCommand handles the RETR command from the client.
func (s *Session) RetrieveCommand(cmd, arg string) error {

	// Close the data connection
	defer s.CloseDataConnection()
	// At this point, dataConn is ready for use for data transfer
	// You can now send or receive data over dataConn
	fmt.Fprintf(s.writer, "150 Opening data connection.\n")
	// Wait for the client to connect on this new port
	dataConn, err := s.PassiveOrActiveModeConn()
	if err != nil {
		fmt.Fprintf(s.writer, "425 Can't open data connection: %s\r\n", err.Error())
		return nil
	}
	defer dataConn.Close()
	filename := Abs(s.root, s.workingDir, arg)
	fmt.Println("RETR:", filename)
	_, err = s.ftpServer.FsHandler.Read(filename, dataConn)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error reading the file: %s\r\n", err.Error())
		return nil
	}

	fmt.Fprintf(s.writer, "226 Transfer complete\r\n")
	return nil
}

func (s *Session) RemoveCommand(cmd, arg string) error {
	fileName := Abs(s.root, s.workingDir, arg)
	err := s.ftpServer.FsHandler.Remove(fileName)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error deleting file: %s\n", err.Error())
		return nil
	}
	fmt.Fprintf(s.writer, "250 File deleted.\r\n")
	return nil
}

func (s *Session) RenameFromCommand(cmd, arg string) error {
	//error reanming file
	if arg == "" {
		fmt.Fprintf(s.writer, "503 No file specified\r\n")
		return nil
	}
	renamingFile := Abs(s.root, s.workingDir, arg)

	_, _, err := s.ftpServer.FsHandler.Stat(renamingFile)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error getting file info: %s\r\n", err.Error())
		return nil
	}
	s.renamingFile = renamingFile

	fmt.Fprintf(s.writer, "350 File exists, ready for destination name\r\n")
	return nil

}

func (s *Session) RenameToCommand(cmd, arg string) error {
	//error reanming file
	if arg == "" {
		fmt.Fprintf(s.writer, "503 No file specified\r\n")
		return nil
	}

	newFileName := Abs(s.root, s.workingDir, arg)

	err := s.ftpServer.FsHandler.Rename(s.renamingFile, newFileName)
	if err != nil {
		fmt.Fprintf(s.writer, "550 Error renaming file: %s\r\n", err.Error())
		return nil
	}
	fmt.Fprintf(s.writer, "250 File renamed successfully.\r\n")

	return nil

}
func (s *Session) CloseCommand(cmd, arg string) error {
	fmt.Fprintf(s.writer, "221 Goodbye.\r\n")
	return nil
}
func (s *Session) UnknownCommand(cmd, arg string) error {
	fmt.Fprintf(s.writer, "500 Unknown command. %s %s\r\n", cmd, arg)
	return nil
}
