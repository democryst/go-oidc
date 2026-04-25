package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/democryst/go-oidc/pkg/interfaces"
)

// --- Rate Limiting ---

type RateLimitStore interface {
	Check(key string, limit int, window time.Duration) bool
}

type MemoryStore struct {
	mu    sync.Mutex
	data  map[string]limitEntry
}

type limitEntry struct {
	count   int
	resetAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]limitEntry)}
}

func (m *MemoryStore) Check(key string, limit int, window time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	entry, exists := m.data[key]

	if !exists || now.After(entry.resetAt) {
		m.data[key] = limitEntry{count: 1, resetAt: now.Add(window)}
		return true
	}

	if entry.count >= limit {
		return false
	}

	entry.count++
	m.data[key] = entry
	return true
}

func RateLimit(store RateLimitStore, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				ip = strings.Split(xff, ",")[0]
			}

			// In OIDC, client_id might be in body or header. 
			// For middleware, we'll try header first, then common body extraction if possible.
			clientID := r.URL.Query().Get("client_id")
			if clientID == "" {
				clientID = r.PostFormValue("client_id")
			}

			key := fmt.Sprintf("%s:%s:%s", r.URL.Path, ip, clientID)
			if !store.Check(key, limit, window) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate_limit_exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// --- Logging ---

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		rw := &responseWriter{w, http.StatusOK}
		// Inject into context
		ctx := context.WithValue(r.Context(), interfaces.RequestIDKey, requestID)
		next.ServeHTTP(rw, r.WithContext(ctx))

		duration := time.Since(start)

		// Mitigation for G706: Sanitize untrusted input
		safeID := strings.ReplaceAll(requestID, "\n", "")
		safeID = strings.ReplaceAll(safeID, "\r", "")
		safeMethod := strings.ReplaceAll(r.Method, "\n", "")
		safeMethod = strings.ReplaceAll(safeMethod, "\r", "")
		safePath := strings.ReplaceAll(r.URL.Path, "\n", "")
		safePath = strings.ReplaceAll(safePath, "\r", "")

		log.Printf("[%s] %s %s | %d | %v | IP: %s", // #nosec G706
			safeID,
			safeMethod,
			safePath,
			rw.status,
			duration,
			r.RemoteAddr,
		)
	})
}
