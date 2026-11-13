package middlewares

import (
	"aaa_bekend/internal/auth"
	"aaa_bekend/internal/lmodel"
	"net/http"
	"strings"
)

func AuthN(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// TODO
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenStr == "" {
			http.Error(w, "Unauthorized: missing or invalid token", http.StatusUnauthorized)
			return
		}

		// TODO
		claims, err := auth.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		userCtx := lmodel.UserContext{ID: claims.UserID, Username: claims.Username, Role: claims.Role}

		ctx := lmodel.ContextWithUser(r.Context(), userCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
