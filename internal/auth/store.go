package auth

import (
	"fmt"
	"sync"
	"time"
)

// AuthCode represents an OAuth authorization code.
type AuthCode struct {
	CodeHash      string
	ClientID      string
	RedirectURI   string
	CodeChallenge string
	Scope         string
	ExpiresAt     time.Time
	Used          bool
}

// StoredToken represents a persisted OAuth token.
type StoredToken struct {
	TokenHash string
	TokenType string // "access" or "refresh"
	ClientID  string
	Scope     string
	ExpiresAt time.Time
	Revoked   bool
}

// AuthStore is the persistence interface for OAuth codes and tokens.
type AuthStore interface {
	StoreCode(code *AuthCode)
	ConsumeCode(codeHash string) (*AuthCode, error)
	StoreToken(token *StoredToken)
	GetToken(tokenHash string) (*StoredToken, error)
	RevokeToken(tokenHash string)
	Cleanup()
}

// MemoryAuthStore handles in-memory persistence of OAuth codes and tokens.
type MemoryAuthStore struct {
	mu     sync.RWMutex
	codes  map[string]*AuthCode
	tokens map[string]*StoredToken
}

// NewMemoryAuthStore creates an in-memory AuthStore.
func NewMemoryAuthStore() *MemoryAuthStore {
	return &MemoryAuthStore{
		codes:  make(map[string]*AuthCode),
		tokens: make(map[string]*StoredToken),
	}
}

func (s *MemoryAuthStore) StoreCode(code *AuthCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code.CodeHash] = code
}

func (s *MemoryAuthStore) ConsumeCode(codeHash string) (*AuthCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	code, ok := s.codes[codeHash]
	if !ok {
		return nil, fmt.Errorf("authorization code not found")
	}
	if code.Used {
		return nil, fmt.Errorf("authorization code already used")
	}
	if time.Now().After(code.ExpiresAt) {
		return nil, fmt.Errorf("authorization code expired")
	}

	code.Used = true
	return code, nil
}

func (s *MemoryAuthStore) StoreToken(token *StoredToken) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token.TokenHash] = token
}

func (s *MemoryAuthStore) GetToken(tokenHash string) (*StoredToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.tokens[tokenHash]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}
	if token.Revoked {
		return nil, fmt.Errorf("token revoked")
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}
	return token, nil
}

func (s *MemoryAuthStore) RevokeToken(tokenHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if token, ok := s.tokens[tokenHash]; ok {
		token.Revoked = true
	}
}

func (s *MemoryAuthStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for hash, code := range s.codes {
		if now.After(code.ExpiresAt) {
			delete(s.codes, hash)
		}
	}
	for hash, token := range s.tokens {
		if now.After(token.ExpiresAt) || token.Revoked {
			delete(s.tokens, hash)
		}
	}
}
