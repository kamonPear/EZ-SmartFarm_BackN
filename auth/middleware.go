package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey int

const (
	userIDContextKey contextKey = iota
	usernameContextKey
	roleContextKey
)

// writeJSONError writes a JSON error body in the shape {"error": "..."} used across
// the auth contract for 401/403 responses.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// RequireAuth validates the Authorization: Bearer <token> header on the request,
// and on success stashes the user's id/username/role into the request context before
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
		ctx = context.WithValue(ctx, roleContextKey, claims.Role)

		next(w, r.WithContext(ctx))
	}
}

// RequireAdmin runs RequireAuth first, then only allows the request through if the
// authenticated user's role is "admin". Otherwise responds 403 {"error":"forbidden"}.
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		role, ok := RoleFromContext(r)
		if !ok || role != "admin" {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
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

// RoleFromContext returns the authenticated user's role, as stashed by RequireAuth.
func RoleFromContext(r *http.Request) (string, bool) {
	v, ok := r.Context().Value(roleContextKey).(string)
	return v, ok
}
