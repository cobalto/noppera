package middleware

import (
	"net/http"
	"os"
	"time"

	"github.com/cobalto/noppera/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logging creates middleware to log HTTP requests and responses.
func Logging(cfg config.Config) func(http.Handler) http.Handler {
	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(parseLogLevel(cfg.LogLevel))
	if cfg.LogFile != "stdout" {
		file, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to open log file")
		}
		defer file.Close()
		log.Logger = log.Output(file).With().Timestamp().Logger()
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			defer func() {
				if rec := recover(); rec != nil {
					log.Error().
						Interface("recover", rec).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Str("remote_addr", r.RemoteAddr).
						Msg("Panic recovered in HTTP handler")
					http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					ww.statusCode = http.StatusInternalServerError
				}

				duration := time.Since(start)
				log.Info().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Int("status", ww.statusCode).
					Int64("duration_ms", duration.Milliseconds()).
					Str("remote_addr", r.RemoteAddr).
					Msg("HTTP request")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture and expose the HTTP status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// parseLogLevel converts string log level to zerolog.Level.
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
