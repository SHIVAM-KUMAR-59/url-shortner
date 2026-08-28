package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/utils"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		apiKey := strings.TrimPrefix(authHeader, "Bearer ")
		if apiKey == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		apiKeyHash := utils.HashAPIKey(apiKey)

		user, err := h.service.GetUserByAPIKeyHash(
			r.Context(),
			apiKeyHash,
		)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(
			r.Context(),
			userIDContextKey,
			user.ID,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})

}
