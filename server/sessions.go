package server

import (
	"github.com/telebroad/ftpserver/users"
	"io"
	"net"
	"sync"
)

// FTPSession represents an individual client FTP session.
type FTPSession struct {
	ftpServer                  *FTPServer   // The server the session belongs to
	conn                       net.Conn     // The connection to the client
	writer                     io.Writer    // Writer for the connection (used for writing responses)
	userInfo                   *users.User  // Authenticated user
	workingDir                 string       // Current working directory
	root                       string       // directory on the server acts as the root
	isAuthenticated            bool         // Authentication status
	dataListener               net.Listener // data transfer connection
	dataListenerPortRangeStart int          // data transfer connection port range
	dataListenerPortRangeEnd   int          // data transfer connection port range
}

// FTPSessionManager manages all active sessions.
type FTPSessionManager struct {
	sessions map[string]*FTPSession // Map of active sessions
	lock     sync.RWMutex           // Protects the sessions map
}

func NewSessionManager() *FTPSessionManager {
	return &FTPSessionManager{
		sessions: make(map[string]*FTPSession),
	}
}

// Add adds a new session for the client.
func (manager *FTPSessionManager) Add(id string, session *FTPSession) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.sessions[id] = session
}

// Get retrieves a session by its ID.
func (manager *FTPSessionManager) Get(id string) (*FTPSession, bool) {
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	session, exists := manager.sessions[id]
	return session, exists
}

// Remove removes a session by its ID.
func (manager *FTPSessionManager) Remove(id string) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	delete(manager.sessions, id)
}
