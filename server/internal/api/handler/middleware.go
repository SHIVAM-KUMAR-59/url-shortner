package handler

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/apperrors"
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

func (h *Handler) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const (
			authenticatedLimit int64 = 100
			anonymousLimit     int64 = 10
			window                   = time.Minute
		)

		var (
			key   string
			limit int64
		)

		if userID := r.Context().Value(userIDContextKey); userID != nil {
			id, ok := userID.(int64)
			if ok {
				key = fmt.Sprintf("ratelimit:user:%d", id)
				limit = authenticatedLimit
			}
		}

		if key == "" {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				// Fallback if RemoteAddr doesn't contain a port.
				host = r.RemoteAddr
			}

			key = fmt.Sprintf("ratelimit:ip:%s", host)
			limit = anonymousLimit
		}

		allowed, err := h.limiter.Allow(
			r.Context(),
			key,
			limit,
			window,
		)
		if err != nil {
			// Fail open: Redis/rate limiter failure should not block the API.
			log.Printf("rate limiter failed for key %s: %v", key, err)

			next.ServeHTTP(w, r)
			return
		}

		if !allowed {
			writeError(
				w,
				apperrors.TooManyRequestsError("rate limit exceeded"),
			)
			return
		}

		next.ServeHTTP(w, r)
	})

}
