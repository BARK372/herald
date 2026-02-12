package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSecret = []byte("test-secret-key-for-signing-tokens")

func TestSignAndVerify_RoundTrip(t *testing.T) {
	t.Parallel()

	claims := TokenClaims{
		Subject:   "user",
		ClientID:  "client-1",
		Scope:     "tasks:write",
		TokenType: "access",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Issuer:    "https://herald.test",
	}

	token, err := SignToken(claims, testSecret)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	verified, err := VerifyToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, "user", verified.Subject)
	assert.Equal(t, "client-1", verified.ClientID)
	assert.Equal(t, "tasks:write", verified.Scope)
	assert.Equal(t, "access", verified.TokenType)
	assert.Equal(t, "https://herald.test", verified.Issuer)
}

func TestVerifyToken_RejectsExpired(t *testing.T) {
	t.Parallel()

	claims := TokenClaims{
		Subject:   "user",
		TokenType: "access",
		IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}

	token, err := SignToken(claims, testSecret)
	require.NoError(t, err)

	_, err = VerifyToken(token, testSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestVerifyToken_RejectsWrongSecret(t *testing.T) {
	t.Parallel()

	claims := TokenClaims{
		Subject:   "user",
		TokenType: "access",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := SignToken(claims, testSecret)
	require.NoError(t, err)

	_, err = VerifyToken(token, []byte("wrong-secret"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature")
}

func TestVerifyToken_RejectsMalformedToken(t *testing.T) {
	t.Parallel()

	_, err := VerifyToken("not.a.valid.token", testSecret)
	require.Error(t, err)
}

func TestVerifyToken_RejectsTamperedPayload(t *testing.T) {
	t.Parallel()

	claims := TokenClaims{
		Subject:   "user",
		TokenType: "access",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := SignToken(claims, testSecret)
	require.NoError(t, err)

	// Tamper with a character in the payload section
	tampered := token[:len(token)/2] + "X" + token[len(token)/2+1:]
	_, err = VerifyToken(tampered, testSecret)
	require.Error(t, err)
}

func TestHashToken_IsDeterministic(t *testing.T) {
	t.Parallel()

	h1 := HashToken("my-token")
	h2 := HashToken("my-token")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64, "SHA-256 hex should be 64 chars")
}

func TestHashToken_DifferentInputs(t *testing.T) {
	t.Parallel()

	h1 := HashToken("token-a")
	h2 := HashToken("token-b")
	assert.NotEqual(t, h1, h2)
}
