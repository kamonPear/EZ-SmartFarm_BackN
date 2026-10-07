package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeJSONErr(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// LoginHandler authenticates a username/password and issues a JWT.
// POST /api/auth/login (public)
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		writeJSONErr(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, err := database.GetUserByUsername(req.Username)
	if err != nil {
		log.Printf("LoginHandler: error fetching user: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Generic 401 for "no such user", "wrong password", and "username matched only
	// because the username column's collation is case-insensitive" - never reveal
	// which. MySQL's default collation (utf8mb4_general_ci) means `WHERE username = ?`
	// above already matched "Kkk" against a stored "kkk" - add an exact Go-side
	// comparison so login genuinely requires the right case, not just the right
	// letters (confirmed live: logging in as "Kkk" succeeded against account "kkk").
	if user == nil || user.Username != req.Username || !auth.CheckPassword(user.Password, req.Password) {
		writeJSONErr(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		log.Printf("LoginHandler: error generating token: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// RegisterHandler creates a new user. Mounted behind auth.RequireAdminKey.
// POST /api/auth/register
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)

	if req.Username == "" || req.Password == "" {
		writeJSONErr(w, http.StatusBadRequest, "username and password are required")
		return
	}

	exists, err := database.UsernameExists(req.Username)
	if err != nil {
		log.Printf("RegisterHandler: error checking username: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if exists {
		writeJSONErr(w, http.StatusConflict, "username already taken")
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("RegisterHandler: error hashing password: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	user := &models.User{
		Username: req.Username,
		Password: hashed,
	}
	if err := database.CreateUser(user); err != nil {
		log.Printf("RegisterHandler: error creating user: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	log.Printf("✓ Registered new user %q", user.Username)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	})
}

// DeleteUserHandler permanently deletes an account and everything it owns (coops and
// everything under them, farm threshold/layout, food stock/imports, etc.) Gated by
// the same X-Admin-Key as RegisterHandler, not auth.RequireAuth - deleting an account
// is an administrative action, not something a logged-in user does to themselves via
// their own token, and this intentionally allows deleting an account by id without
// needing to log in as that account first (e.g. the account's credentials were lost).
// DELETE /api/auth/users?id=<id>  needs X-Admin-Key
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeJSONErr(w, http.StatusBadRequest, "missing id parameter")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "invalid id parameter")
		return
	}

	if _, err := database.GetUserByID(id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			writeJSONErr(w, http.StatusNotFound, "user not found")
			return
		}
		log.Printf("DeleteUserHandler: error looking up user %d: %v", id, err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := database.DeleteUser(id); err != nil {
		log.Printf("DeleteUserHandler: error deleting user %d: %v", id, err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	log.Printf("✓ Deleted user %d and all owned data", id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "user and all owned data deleted"})
}

// MeHandler returns the authenticated caller's own user record. Mounted behind auth.RequireAuth.
// GET /api/auth/me
func MeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := database.GetUserByID(userID)
	if err != nil {
		writeJSONErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	})
}
