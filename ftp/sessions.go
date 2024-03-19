package ftp

import (
	"github.com/telebroad/fileserver/tools"
	"net"
	"sync"
)

// Session represents an individual client FTP session.
type Session struct {
	ftpServer                  *Server                 // The server the session belongs to
	conn                       net.Conn                // The connection to the client
	readWriter                 *tools.BufLogReadWriter // ReadWriter for the connection (used for writing responses)
	userInfo                   any                     // Authenticated user
	workingDir                 string                  // Current working directory
	root                       string                  // directory on the server acts as the root
	username                   string                  // Username of the client
	isAuthenticated            bool                    // Authentication status
	useTLSForDataConnection    bool                    // Data listener level false is `C` clear, if true is `P` protected
	dataListener               net.Listener            // data transfer connection
	dataCaller                 net.Conn                // data transfer connection
	dataListenerPortRangeStart int                     // data transfer connection port range
	dataListenerPortRangeEnd   int                     // data transfer connection port range
	renamingFile               string                  // File to be renamed
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
