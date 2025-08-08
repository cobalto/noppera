package middleware

import (
	"net/http"
	"time"

	"github.com/cobalto/noppera/internal/config"
	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
)

// RateLimit creates middleware to limit requests based on config.
func RateLimit(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Configure limiter: requests per second with burst
		lim := tollbooth.NewLimiter(float64(cfg.RateLimitRequests)/3600.0, &limiter.ExpirableOptions{
			DefaultExpirationTTL: 3600 * time.Second,
		}).SetBurst(cfg.RateLimitBurst).SetIPLookups([]string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"})

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Apply rate-limiting
			httpError := tollbooth.LimitByRequest(lim, w, r)
			if httpError != nil {
				http.Error(w, httpError.Message, httpError.StatusCode)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitPublic applies rate-limiting to public endpoints.
func RateLimitPublic(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return RateLimit(cfg)(next)
	}
}
