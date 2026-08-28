package handler

import (
	"encoding/json"
	"net/http"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/api/service"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/apperrors"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/ratelimit"
)

type Handler struct {
	service *service.Service
	limiter ratelimit.Limiter
}

func NewHandler(service *service.Service, limiter ratelimit.Limiter) *Handler {
	return &Handler{
		service: service,
		limiter: limiter,
	}
}

type shortenRequest struct {
	LongURL string `json:"long_url"`
}

type shortenResponse struct {
	ShortCode string `json:"short_code"`
}

func (h *Handler) HandleShorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperrors.ValidationError("invalid request body"))
		return
	}

	var userID *int64

	if v := r.Context().Value(userIDContextKey); v != nil {
		if id, ok := v.(int64); ok {
			userID = &id
		}
	}

	shortCode, err := h.service.CreateShortURL(
		r.Context(),
		req.LongURL,
		userID,
	)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(shortenResponse{
		ShortCode: shortCode,
	})

}

func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	shortCode := r.PathValue("short_code")

	longURL, err := h.service.GetLongURL(r.Context(), shortCode)
	if err != nil {
		writeError(w, err)
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)

}

type createUserRequest struct {
	Email string `json:"email"`
}

type createUserResponse struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	APIKey string `json:"api_key"`
}

func (h *Handler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperrors.ValidationError("invalid request body"))
		return
	}

	user, apiKey, err := h.service.CreateUser(r.Context(), req.Email)
	if err != nil {
		writeError(w, err)
		return
	}

	response := createUserResponse{
		UserID: user.ID,
		Email:  user.Email,
		APIKey: apiKey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(response)

}
