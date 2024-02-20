package ftp

import (
	"bufio"
	"fmt"
	"io"
)

type fileServer struct {
}

type LogWriter struct {
	io.Writer
}

func (w *LogWriter) Write(b []byte) (int, error) {
	fmt.Println("Responding:", string(b))
	return w.Writer.Write(b)
}

func (h fileServer) ServeFTP(w ResponseWriter, r *Request) {

	defer conn.Close()
	logWriter := &LogWriter{w}
	sessionID := generateSessionID(conn)
	session := &Session{
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
		switch r.Method {
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

func FileServer() Handler {
	return fileServer{}
}
