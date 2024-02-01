package server

import (
	"bufio"
	"fmt"
	"net"
	"path/filepath"
	"runtime"
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

func (s *FTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	sessionID := generateSessionID(conn)
	session := &Session{
		conn:            conn,
		userInfo:        nil,
		workingDir:      "/", // Set the initial working directory
		isAuthenticated: false,
		root:            s.root,
	}
	ftpSession := &FTPSession{
		Session:   session,
		ftpServer: s,
	}
	// Add the session to the manager
	s.sessionManager.Add(sessionID, session)

	// Example: Authenticate the user
	authenticateUser(session)

	// Remove the session when the client disconnects
	defer s.sessionManager.Remove(sessionID)

	reader := bufio.NewReader(conn)
	// Send a welcome message
	fmt.Fprintln(conn, "220", s.WelcomeMessage)

	for {

		cmd, arg, err := s.ParseCommand(reader)
		if err != nil {
			fmt.Fprintln(conn, err.Error())
			return
		}
		// Handle commands
		switch cmd {
		case "USER":
			resp, err := ftpSession.UserCommand(arg)
			if err != nil {
				fmt.Fprintln(conn, err.Error())
				return
			}
			fmt.Fprintln(conn, resp)
		case "PASS":
			resp, err := ftpSession.PassCommand(arg)
			if err != nil {
				fmt.Fprintln(conn, err.Error())
				return
			}
			fmt.Fprintln(conn, resp)
		// Add more cases here for other commands
		case "SYST":
			fmt.Fprintln(conn, ftpSession.SystemCommand())
		case "FEAT":
			fmt.Fprintln(conn, ftpSession.FeaturesCommand())

		case "PWD":
			fmt.Fprintln(conn, ftpSession.PrintWorkingDirectoryCommand())
		case "CWD":
			resp, err := ftpSession.ChangeDirectoryCommand(arg)
			if err != nil {
				fmt.Fprintln(conn, err.Error())
				return
			}
			fmt.Fprintln(conn, resp)

		case "REST":
			if arg == "0" {
				fmt.Fprintln(conn, "350 Ready for file transfer.")
			} else {
				fmt.Fprintln(conn, "350 Restarting at "+arg+". Send STORE or RETRIEVE.")
			}
		case "TYPE":
			if arg == "I" {
				fmt.Fprintln(conn, "200 Type set to I")
			} else if arg == "A" {
				fmt.Fprintln(conn, "200 Type set to A")
			} else {
				fmt.Fprintln(conn, "500 Unknown type.")
			}

		case "PASV":
			fmt.Fprintln(conn, "227 Entering Passive Mode (h1,h2,h3,h4,p1,p2)")
		case "LIST":
			fmt.Fprintln(conn, "150 Here comes the directory listing.")
			// Send the directory listing
			entries, err := s.fs.Dir("/")
			if err != nil {
				fmt.Fprintln(conn, "550 Error getting directory listing.")
				break
			}
			for _, entry := range entries {
				fmt.Fprintln(conn, entry)
			}
			fmt.Fprintln(conn, "226 Directory send OK.")
		case "RETR":
			fmt.Fprintln(conn, "150 Opening data connection.")
		case "EPSV":
			fmt.Fprintln(conn, "229 Entering Extended Passive Mode (|||6446|)")
			// Send the file
		case "QUIT":
			fmt.Fprintln(conn, "221 Goodbye.")
			return
		default:
			fmt.Fprintln(conn, "500 Unknown command.")
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

type FTPSession struct {
	*Session
	ftpServer *FTPServer
}

// UserCommand handles the USER command from the client.
func (s *FTPSession) UserCommand(arg string) (resp string, err error) {
	if arg == "" {
		return "", fmt.Errorf("530 Error: User name not specified")
	}
	user, err := s.ftpServer.users.Get(arg)
	if err != nil {
		return "", fmt.Errorf("530 Error: Searching for user failed")
	}
	s.userInfo = user
	return "331 Please specify the password", nil
}

// PassCommand handles the PASS command from the client.
func (s *FTPSession) PassCommand(arg string) (resp string, err error) {
	if s.userInfo == nil {
		return "", fmt.Errorf("430 Invalid username or password")
	}
	if s.userInfo.Password != arg {
		return "", fmt.Errorf("430 Invalid username or password")
	}
	return "230 Login successful", nil
}

// SystemCommand returns the system type.
func (s *FTPSession) SystemCommand() string {
	// Use runtime.GOOS to get the operating system name
	os := runtime.GOOS

	// Customize the response based on the operating system
	switch os {
	case "windows":
		return "215 WINDOWS Type: L8"
	case "linux", "darwin":
		return "215 UNIX Type: L8" // macOS is Unix-based
	default:
		return "215 OS Type: " + os
	}
}

func (s *FTPSession) FeaturesCommand() string {
	f := []string{
		"211-Features:",
		" UTF8",
		" MLST type*;size*;modify*;",
		" MLSD",
		" SIZE",
		" MDTM",
		" REST STREAM",
		//" TVFS",
		//" EPSV",
		//" EPRT",
	}

	if s.ftpServer.supportsTLS {
		f = append(f,
			" AUTH TLS",
			" AUTH SSL",
			" PBSZ",
			" PROT",
		)
	}
	f = append(f, "211 End")
	return strings.Join(f, "\n")
}

func (s *FTPSession) PrintWorkingDirectoryCommand() string {
	return fmt.Sprintf("257 \"%s\" is the current directory.", s.workingDir)
}
func (s *FTPSession) ChangeDirectoryCommand(arg string) (res string, err error) {
	// Resolve the requested directory to an absolute path
	requestedDir := ""
	if filepath.IsAbs(arg) {
		requestedDir = filepath.Join(s.ftpServer.fs.RootDir(), arg[1:])
	} else {
		requestedDir = filepath.Join(s.workingDir, arg)
	}

	//requestedDir = filepath.Clean(requestedDir)
	// if after the request is joined with the absolute path, the result is ".." then return an error
	if strings.HasPrefix(requestedDir, "..") {
		return "", fmt.Errorf("550 Failed to change directory")
	}

	err = s.ftpServer.fs.CheckDir(requestedDir)
	if err != nil {
		return "", fmt.Errorf("550 Error: %w", err)
	}

	s.workingDir = requestedDir

	return fmt.Sprintf("250 Directory successfully changed to \"%s\"", requestedDir), nil
}
