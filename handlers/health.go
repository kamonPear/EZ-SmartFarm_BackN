package handlers

import (
	"log"
	"net/http"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Server is running", r.Method, r.RequestURI, http.StatusOK)
	w.Write([]byte(`{"status": "ok", "message": "Server is running"}`))
}
