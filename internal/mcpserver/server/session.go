package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// MCPSession represents an active MCP client connection
type MCPSession struct {
	ID        string
	UserID    string // From JWT sub claim
	CreatedAt time.Time
	LastSeen  time.Time
	// SSE connection tracking would go here in future
}

// SessionManager manages MCP sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*MCPSession // sessionID -> session
	ttl      time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(ttl time.Duration) *SessionManager {
	mgr := &SessionManager{
		sessions: make(map[string]*MCPSession),
		ttl:      ttl,
	}

	// Start cleanup goroutine
	go mgr.cleanupExpired()

	return mgr
}

// CreateSession creates a new MCP session for a user
func (sm *SessionManager) CreateSession(userID string) *MCPSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &MCPSession{
		ID:        uuid.New().String(),
		UserID:    userID,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	sm.sessions[session.ID] = session

	log.Debug().
		Str("sessionId", session.ID).
		Str("userId", userID).
		Msg("Created MCP session")

	return session
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*MCPSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	return session, nil
}

// UpdateLastSeen updates the last seen time for a session
func (sm *SessionManager) UpdateLastSeen(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.LastSeen = time.Now()
	}
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, sessionID)

	log.Debug().
		Str("sessionId", sessionID).
		Msg("Deleted MCP session")
}

// cleanupExpired removes expired sessions
func (sm *SessionManager) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		expired := 0
		for id, session := range sm.sessions {
			if now.Sub(session.LastSeen) > sm.ttl {
				delete(sm.sessions, id)
				expired++
			}
		}
		sm.mu.Unlock()

		if expired > 0 {
			log.Info().
				Int("count", expired).
				Msg("Cleaned up expired MCP sessions")
		}
	}
}
