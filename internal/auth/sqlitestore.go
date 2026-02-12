package auth

import (
	"log/slog"
	"time"

	"github.com/kolapsis/herald/internal/store"
)

// SQLiteAuthStore adapts store.Store to the AuthStore interface.
type SQLiteAuthStore struct {
	db store.Store
}

// NewSQLiteAuthStore creates an AuthStore backed by SQLite.
func NewSQLiteAuthStore(db store.Store) *SQLiteAuthStore {
	return &SQLiteAuthStore{db: db}
}

func (s *SQLiteAuthStore) StoreCode(code *AuthCode) {
	err := s.db.StoreAuthCode(&store.AuthCodeRecord{
		CodeHash:      code.CodeHash,
		ClientID:      code.ClientID,
		RedirectURI:   code.RedirectURI,
		CodeChallenge: code.CodeChallenge,
		Scope:         code.Scope,
		ExpiresAt:     code.ExpiresAt,
		Used:          code.Used,
		CreatedAt:     time.Now(),
	})
	if err != nil {
		slog.Error("failed to store auth code", "error", err)
	}
}

func (s *SQLiteAuthStore) ConsumeCode(codeHash string) (*AuthCode, error) {
	rec, err := s.db.ConsumeAuthCode(codeHash)
	if err != nil {
		return nil, err
	}
	return &AuthCode{
		CodeHash:      rec.CodeHash,
		ClientID:      rec.ClientID,
		RedirectURI:   rec.RedirectURI,
		CodeChallenge: rec.CodeChallenge,
		Scope:         rec.Scope,
		ExpiresAt:     rec.ExpiresAt,
		Used:          rec.Used,
	}, nil
}

func (s *SQLiteAuthStore) StoreToken(token *StoredToken) {
	err := s.db.StoreToken(&store.TokenRecord{
		TokenHash: token.TokenHash,
		TokenType: token.TokenType,
		ClientID:  token.ClientID,
		Scope:     token.Scope,
		ExpiresAt: token.ExpiresAt,
		Revoked:   token.Revoked,
		CreatedAt: time.Now(),
	})
	if err != nil {
		slog.Error("failed to store token", "error", err)
	}
}

func (s *SQLiteAuthStore) GetToken(tokenHash string) (*StoredToken, error) {
	rec, err := s.db.GetToken(tokenHash)
	if err != nil {
		return nil, err
	}
	return &StoredToken{
		TokenHash: rec.TokenHash,
		TokenType: rec.TokenType,
		ClientID:  rec.ClientID,
		Scope:     rec.Scope,
		ExpiresAt: rec.ExpiresAt,
		Revoked:   rec.Revoked,
	}, nil
}

func (s *SQLiteAuthStore) RevokeToken(tokenHash string) {
	if err := s.db.RevokeToken(tokenHash); err != nil {
		slog.Error("failed to revoke token", "error", err)
	}
}

func (s *SQLiteAuthStore) Cleanup() {
	if err := s.db.Cleanup(); err != nil {
		slog.Error("failed to cleanup expired tokens", "error", err)
	}
}
