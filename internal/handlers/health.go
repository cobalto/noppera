package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterHealth sets up health check routes.
func RegisterHealth(r chi.Router, db *pgxpool.Pool) {
	r.Get("/health", healthCheck(db))
	r.Get("/health/ready", readinessCheck(db))
	r.Get("/health/live", livenessCheck())
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services,omitempty"`
	Uptime    string            `json:"uptime,omitempty"`
}

var startTime = time.Now()

// healthCheck handles GET /health, basic health check.
// @Summary Health check
// @Description Check the health status of the API and its dependencies
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse "Service is healthy"
// @Failure 503 {object} HealthResponse "Service is unhealthy"
// @Router /health [get]
func healthCheck(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		services := make(map[string]string)

		// Check database connectivity
		if err := db.Ping(ctx); err != nil {
			services["database"] = "unhealthy"
		} else {
			services["database"] = "healthy"
		}

		status := "healthy"
		for _, serviceStatus := range services {
			if serviceStatus != "healthy" {
				status = "unhealthy"
				break
			}
		}

		response := HealthResponse{
			Status:    status,
			Timestamp: time.Now(),
			Services:  services,
			Uptime:    time.Since(startTime).String(),
		}

		w.Header().Set("Content-Type", "application/json")
		if status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(response)
	}
}

// readinessCheck handles GET /health/ready, readiness probe for Kubernetes.
// @Summary Readiness check
// @Description Check if the service is ready to accept traffic
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string "Service is ready"
// @Failure 503 {object} map[string]string "Service is not ready"
// @Router /health/ready [get]
func readinessCheck(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Check if database is ready
		if err := db.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "not ready",
				"error":  err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	}
}

// livenessCheck handles GET /health/live, liveness probe for Kubernetes.
// @Summary Liveness check
// @Description Check if the service is alive
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string "Service is alive"
// @Router /health/live [get]
func livenessCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
		})
	}
}
