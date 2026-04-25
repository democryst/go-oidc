package middleware

import (
	"net/http"
	"os"
	"strings"
)

// AdminAuth validates the Bearer token against the ADMIN_API_KEY environment variable.
func AdminAuth(next http.Handler) http.Handler {
	apiKey := os.Getenv("ADMIN_API_KEY")
	if apiKey == "" {
		// If not configured, we fail-closed for security.
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Admin API key not configured", http.StatusInternalServerError)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
