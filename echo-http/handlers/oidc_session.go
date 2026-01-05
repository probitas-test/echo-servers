package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session represents an OIDC session
type Session struct {
	ID          string
	State       string // Client-provided state (optional, may be empty)
	RedirectURI string
	Scope       string
	CreatedAt   time.Time
}

// AuthCode represents an authorization code issued after authentication
type AuthCode struct {
	Code        string
	RedirectURI string
	Username    string
	Scope       string
	CreatedAt   time.Time
}

// SessionStore provides in-memory storage for OIDC sessions and authorization codes
type SessionStore struct {
	sessions  map[string]*Session // key = session ID
	authCodes map[string]*AuthCode
	mu        sync.RWMutex
	ttl       time.Duration
}

var (
	// DefaultSessionStore is the global session store instance
	DefaultSessionStore = NewSessionStore(5 * time.Minute)
)

// NewSessionStore creates a new session store with the given TTL
func NewSessionStore(ttl time.Duration) *SessionStore {
	store := &SessionStore{
		sessions:  make(map[string]*Session),
		authCodes: make(map[string]*AuthCode),
		ttl:       ttl,
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

// CreateSession creates a new session with optional client-provided state
func (s *SessionStore) CreateSession(state, redirectURI, scope string) (*Session, error) {
	sessionID, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:          sessionID,
		State:       state, // Client-provided (may be empty)
		RedirectURI: redirectURI,
		Scope:       scope,
		CreatedAt:   time.Now(),
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by session ID
func (s *SessionStore) GetSession(sessionID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}

	// Check if session is expired
	if time.Since(session.CreatedAt) > s.ttl {
		return nil, false
	}

	return session, true
}

// DeleteSession removes a session by session ID
func (s *SessionStore) DeleteSession(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

// CreateAuthCode creates a new authorization code
func (s *SessionStore) CreateAuthCode(redirectURI, username, scope string) (*AuthCode, error) {
	code, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	authCode := &AuthCode{
		Code:        code,
		RedirectURI: redirectURI,
		Username:    username,
		Scope:       scope,
		CreatedAt:   time.Now(),
	}

	s.mu.Lock()
	s.authCodes[code] = authCode
	s.mu.Unlock()

	return authCode, nil
}

// GetAuthCode retrieves an authorization code
func (s *SessionStore) GetAuthCode(code string) (*AuthCode, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	authCode, ok := s.authCodes[code]
	if !ok {
		return nil, false
	}

	// Check if auth code is expired
	if time.Since(authCode.CreatedAt) > s.ttl {
		return nil, false
	}

	return authCode, true
}

// DeleteAuthCode removes an authorization code (single-use)
func (s *SessionStore) DeleteAuthCode(code string) {
	s.mu.Lock()
	delete(s.authCodes, code)
	s.mu.Unlock()
}

// cleanup periodically removes expired sessions and auth codes
func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()

		// Clean up expired sessions
		for sessionID, session := range s.sessions {
			if now.Sub(session.CreatedAt) > s.ttl {
				delete(s.sessions, sessionID)
			}
		}

		// Clean up expired auth codes
		for code, authCode := range s.authCodes {
			if now.Sub(authCode.CreatedAt) > s.ttl {
				delete(s.authCodes, code)
			}
		}

		s.mu.Unlock()
	}
}

// generateRandomString generates a cryptographically secure random string
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
