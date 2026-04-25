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
	_ = json.NewEncoder(w).Encode(stats)
}

func (h *AdminHandler) HandleAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Querying the Postgres audit_log table.
	ctx := r.Context()
	logs, err := h.svc.GetAuditLogs(ctx, 50) // Need to add this method to service
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(logs)
}

func (h *AdminHandler) HandleClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clients, err := h.svc.ListClients(ctx) // Need to add this method to service
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(clients)
}

func (h *AdminHandler) HandleCreateClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name         string   `json:"name"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	client, err := h.svc.RegisterClient(ctx, req.Name, req.RedirectURIs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(client)
}

func (h *AdminHandler) HandleRotateKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	if err := h.svc.RotatePQCKeys(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
