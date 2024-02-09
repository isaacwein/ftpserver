package ftp

import (
	"github.com/telebroad/ftpserver/users"
	"io"
	"net"
	"sync"
)

// conn represents an individual client FTP session.
type conn struct {
	ftpServer                  *Server      // The server the session belongs to
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
type connManager struct {
	sessions map[string]*conn // Map of active sessions
	lock     sync.RWMutex     // Protects the sessions map
}

func newSessionManager() *connManager {
	return &connManager{
		sessions: make(map[string]*conn),
	}
}

// Add adds a new session for the client.
func (manager *connManager) Add(id string, session *conn) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.sessions[id] = session
}

// Get retrieves a session by its ID.
func (manager *connManager) Get(id string) (*conn, bool) {
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	session, exists := manager.sessions[id]
	return session, exists
}

// Remove removes a session by its ID.
func (manager *connManager) Remove(id string) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	delete(manager.sessions, id)
}
