package handlers

import (
	"encoding/json"
	"log"
	"net/http"
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

	// Generic 401 for both "no such user" and "wrong password" - never reveal which.
	if user == nil || !auth.CheckPassword(user.Password, req.Password) {
		writeJSONErr(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
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
			"role":     user.Role,
		},
	})
}

// RegisterHandler creates a new user. Mounted behind auth.RequireAdmin.
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
	req.Role = strings.TrimSpace(req.Role)

	if req.Username == "" || req.Password == "" {
		writeJSONErr(w, http.StatusBadRequest, "username and password are required")
		return
	}

	role := req.Role
	if role == "" {
		role = "user"
	}
	if role != "user" && role != "admin" {
		writeJSONErr(w, http.StatusBadRequest, "role must be either 'admin' or 'user'")
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
		Role:     role,
	}
	if err := database.CreateUser(user); err != nil {
		log.Printf("RegisterHandler: error creating user: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	log.Printf("✓ Registered new user %q (role=%s)", user.Username, user.Role)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
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
		"role":     user.Role,
	})
}
