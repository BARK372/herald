package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TokenClaims represents the payload of a Herald JWT.
type TokenClaims struct {
	JTI       string `json:"jti"`
	Subject   string `json:"sub"`
	ClientID  string `json:"client_id"`
	Scope     string `json:"scope"`
	TokenType string `json:"token_type"` // "access" or "refresh"
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Issuer    string `json:"iss"`
}

// IsExpired returns true if the token has expired.
func (c TokenClaims) IsExpired() bool {
	return time.Now().Unix() > c.ExpiresAt
}

// jwtHeader is the fixed JWT header for Herald tokens (HMAC-SHA256).
var jwtHeader = base64url([]byte(`{"alg":"HS256","typ":"JWT"}`))

// SignToken creates a signed JWT from the given claims.
// A unique JTI is generated automatically if not set.
func SignToken(claims TokenClaims, secret []byte) (string, error) {
	if claims.JTI == "" {
		claims.JTI = generateJTI()
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshaling claims: %w", err)
	}

	encodedPayload := base64url(payload)
	signingInput := jwtHeader + "." + encodedPayload

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	signature := base64url(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// VerifyToken validates a JWT signature and returns the claims.
func VerifyToken(tokenStr string, secret []byte) (*TokenClaims, error) {
	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	signatureBytes, err := base64urlDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(signatureBytes, expectedSig) {
		return nil, fmt.Errorf("invalid signature")
	}

	payload, err := base64urlDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding: %w", err)
	}

	var claims TokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	if claims.IsExpired() {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// HashToken returns the SHA-256 hex hash of a token (for storage lookup).
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

func base64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64urlDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func generateJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
