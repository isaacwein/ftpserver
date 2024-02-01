package server

import (
	"github.com/telebroad/ftpserver/users"
	"net"
	"sync"
)

// Session represents an individual client FTP session.
type Session struct {
	conn            net.Conn    // The connection to the client
	userInfo        *users.User // Authenticated user
	workingDir      string      // Current working directory
	root            string      // directory on the server acts as the root
	isAuthenticated bool        // Authentication status
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
