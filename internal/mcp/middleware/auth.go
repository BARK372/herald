package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/kolapsis/herald/internal/auth"
)

// BearerAuth returns middleware that validates OAuth 2.1 Bearer tokens.
func BearerAuth(oauth *auth.OAuthServer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				unauthorized(w, "missing Authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				unauthorized(w, "invalid Authorization header format")
				return
			}

			token := parts[1]
			claims, err := oauth.ValidateAccessToken(token)
			if err != nil {
				slog.Debug("token validation failed", "error", err)
				unauthorized(w, "invalid or expired token")
				return
			}

			slog.Debug("request authenticated",
				"client_id", claims.ClientID,
				"scope", claims.Scope)

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	http.Error(w, msg, http.StatusUnauthorized)
}
