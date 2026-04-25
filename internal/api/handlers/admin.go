package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/democryst/go-oidc/pkg/interfaces"
)

type AdminHandler struct {
	svc interfaces.OIDCService
}

func NewAdminHandler(svc interfaces.OIDCService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

type StatsResponse struct {
	Timestamp      time.Time `json:"timestamp"`
	TPS            float64   `json:"tps"`
	P99Latency     string    `json:"p99_latency"`
	SuccessRate    float64   `json:"success_rate"`
	ActiveSessions int       `json:"active_sessions"`
}

func (h *AdminHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	// In a real Phase 2/3 implementation, this would pull from Prometheus or Redis.
	// For this sprint, we return mock real-time data to demonstrate the UI.
	stats := StatsResponse{
		Timestamp:      time.Now(),
		TPS:            12450.5,
		P99Latency:     "0.74ms",
		SuccessRate:    99.98,
		ActiveSessions: 45000,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *AdminHandler) HandleAuditLogs(w http.ResponseWriter, r *http.Request) {
	// In production, this would query the Postgres audit_log table.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`[]`))
}

func (h *AdminHandler) HandleClients(w http.ResponseWriter, r *http.Request) {
	// Return registered OIDC clients.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`[]`))
}
