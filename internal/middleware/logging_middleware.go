package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TokenParser interface {
	Parse(string) (int, error)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		request_id := RequestIDFromContext(r.Context())

		slog.Info("request completed",
			"duration", time.Since(t),
			"statusCode", rw.statusCode,
			"path", r.URL.Path,
			"method", r.Method,
			"request_id", request_id)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

type contextKey string

const requestIDKey contextKey = "request_id"

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func RequestIDFromContext(ctx context.Context) string {
	request_id, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}

	return request_id
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"statusCode", http.StatusInternalServerError,
					"request_id", RequestIDFromContext(r.Context()),
					"debug_stack", string(debug.Stack()))

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

const userIDKey contextKey = "user_id"

func AuthMiddleware(next http.Handler, parser TokenParser) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		parts := strings.Fields(auth)
		if len(parts) != 2 {
			jsonError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if !strings.EqualFold(parts[0], "Bearer") {
			jsonError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		token := parts[1]
		userID, err := parser.Parse(token)
		if err != nil {
			jsonError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func UserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userIDKey).(int)
	if userID <= 0 || !ok {
		return 0, false
	}

	return userID, ok
}

func jsonError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
