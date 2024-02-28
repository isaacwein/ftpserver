package ftp

import (
	"bufio"
	"github.com/telebroad/ftpserver/ftp/ftpusers"
	"io"
	"log/slog"
	"net"
	"sync"
)

// Session represents an individual client FTP session.
type Session struct {
	ftpServer                  *Server           // The server the session belongs to
	conn                       net.Conn          // The connection to the client
	readWriter                 *BufLogReadWriter // ReadWriter for the connection (used for writing responses)
	userInfo                   *ftpusers.User    // Authenticated user
	workingDir                 string            // Current working directory
	root                       string            // directory on the server acts as the root
	isAuthenticated            bool              // Authentication status
	useTLSForDataConnection    bool              // Data listener level false is `C` clear, if true is `P` protected
	dataListener               net.Listener      // data transfer connection
	dataCaller                 net.Conn          // data transfer connection
	dataListenerPortRangeStart int               // data transfer connection port range
	dataListenerPortRangeEnd   int               // data transfer connection port range
	renamingFile               string            // File to be renamed
	HelpCommands               string
}

// SessionManager manages all active sessions.
type SessionManager struct {
	sessions map[string]*Session // Map of active sessions
	lock     sync.RWMutex        // Protects the sessions map
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// Add adds a new session for the client.
func (manager *SessionManager) Add(id string, session *Session) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.sessions[id] = session
}

// Get retrieves a session by its ID.
func (manager *SessionManager) Get(id string) (*Session, bool) {
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	session, exists := manager.sessions[id]
	return session, exists
}

// Remove removes a session by its ID.
func (manager *SessionManager) Remove(id string) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	delete(manager.sessions, id)
}

// LogReaderWriter is a wrapper around a bufio.ReadWriter that logs all reads and writes to a slog.Logger.
type LogReaderWriter struct {
	ReadWriter io.ReadWriter
	logger     *slog.Logger
}

func (rw *LogReaderWriter) Read(b []byte) (int, error) {
	n, err := rw.ReadWriter.Read(b)
	if rw.logger != nil && n > 0 { // Log only if n > 0 to avoid logging empty reads
		rw.logger.Debug("Request", "body", string(b[:n])) // Adjusted to log only the read portion
	}
	return n, err
}

func (rw *LogReaderWriter) Write(b []byte) (int, error) {
	if rw.logger != nil {
		rw.logger.Debug("Respond", "body", string(b))
	}
	return rw.ReadWriter.Write(b)
}

// NewLogReadWriter creates a new LogReaderWriter.
func NewLogReadWriter(rw io.ReadWriter, logger *slog.Logger) *LogReaderWriter {
	return &LogReaderWriter{ReadWriter: rw, logger: logger}
}

type BufLogReadWriter struct {
	io.Writer
	*bufio.Reader
}

// NewBufLogReadWriter creates a new BufLogReadWriter. It wraps a bufio.ReadWriter and logs all reads and writes to a slog.Logger.
// the reason to divide it in 2 structs is to avoid the need to implement all the methods of bufio.ReadWriter
func NewBufLogReadWriter(rw io.ReadWriter, logger *slog.Logger) *BufLogReadWriter {
	rw = &LogReaderWriter{ReadWriter: rw, logger: logger}

	return &BufLogReadWriter{
		Reader: bufio.NewReader(rw),
		Writer: rw,
	}
}
