package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

// adminKeyWarn keeps the "ADMIN_API_KEY is not set" warning to one line per process
// instead of one per rejected request.
var adminKeyWarn sync.Once

type contextKey int

const (
	userIDContextKey contextKey = iota
	usernameContextKey
)

// writeJSONError writes a JSON error body in the shape {"error": "..."} used across
// the auth contract for 401/403 responses.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// RequireAuth validates the Authorization: Bearer <token> header on the request,
// and on success stashes the user's id/username into the request context before
// calling next. On a missing/invalid/expired token it responds 401 {"error":"unauthorized"}.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		claims, err := ParseToken(strings.TrimSpace(parts[1]))
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		userID, err := claims.UserID()
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, userIDContextKey, userID)
		ctx = context.WithValue(ctx, usernameContextKey, claims.Username)

		next(w, r.WithContext(ctx))
	}
}

// RequireAdminKey gates endpoints that create accounts. There are no roles in this
// system - every account is just a user - so the gate is a shared secret held by
// whoever administers the farm, sent as the X-Admin-Key header and compared against
// ADMIN_API_KEY from the environment. Without it, anyone who found the deployed URL
// could create themselves an account.
func RequireAdminKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expected := os.Getenv("ADMIN_API_KEY")
		if expected == "" {
			// Fail closed: an unset key must not silently leave the endpoint open.
			adminKeyWarn.Do(func() {
				log.Println("WARNING: ADMIN_API_KEY is not set - /api/auth/register is disabled until it is.")
			})
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		provided := r.Header.Get("X-Admin-Key")
		// Constant-time compare so a wrong key can't be recovered by timing the response.
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		next(w, r)
	}
}

// UserIDFromContext returns the authenticated user's id, as stashed by RequireAuth.
func UserIDFromContext(r *http.Request) (int, bool) {
	v, ok := r.Context().Value(userIDContextKey).(int)
	return v, ok
}

// UsernameFromContext returns the authenticated user's username, as stashed by RequireAuth.
func UsernameFromContext(r *http.Request) (string, bool) {
	v, ok := r.Context().Value(usernameContextKey).(string)
	return v, ok
}
