package middleware

import (
	"net/http"
	"strings"

	"github.com/cobalto/noppera/internal/config"
)

// CORS creates middleware to handle Cross-Origin Resource Sharing.
func CORS(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			origins := strings.Split(cfg.CORSAllowedOrigins, ",")
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			if cfg.CORSAllowedOrigins == "*" {
				allowed = true
			} else {
				for _, allowedOrigin := range origins {
					if strings.TrimSpace(allowedOrigin) == origin {
						allowed = true
						break
					}
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", cfg.CORSAllowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", cfg.CORSAllowedHeaders)

			if cfg.CORSAllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
