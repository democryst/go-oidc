package handlers

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct {
	// Add health-related dependencies here (e.g. DB pool, Redis client)
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	// Liveness: indicates if the app is still running.
	// Minimum check: just return OK.
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	// Readiness: indicates if the app is ready to serve traffic.
	// Check DB and Redis availability here.
	status := map[string]string{
		"status": "UP",
		"db":     "READY",
		"redis":  "READY",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}
