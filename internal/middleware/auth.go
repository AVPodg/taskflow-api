package middleware

import (
	"context"
	"net/http"
	"pgx-practice/internal/httputil"
	"strings"
)

const userIDKey contextKey = "user_id"

type TokenParser interface {
	Parse(string) (int, error)
}

func AuthMiddleware(next http.Handler, parser TokenParser) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		parts := strings.Fields(auth)
		if len(parts) != 2 {
			httputil.JSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if !strings.EqualFold(parts[0], "Bearer") {
			httputil.JSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		token := parts[1]
		userID, err := parser.Parse(token)
		if err != nil {
			httputil.JSONError(w, http.StatusUnauthorized, "unauthorized")
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
