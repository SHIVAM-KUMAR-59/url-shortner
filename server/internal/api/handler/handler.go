package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/api/service"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/apperrors"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var userID *int64
	if v := r.Context().Value(userIDContextKey); v != nil {
		id := v.(int64)
		userID = &id
	}

	shortCode, err := h.service.CreateShortURL(r.Context(), req.LongURL, userID)
	if err != nil {
		statusCode := apperrors.GetStatusCode(err)

		message := "internal server error"

		switch {
		case errors.Is(err, apperrors.ErrInvalidLongURL):
			message = "invalid or missing long url"

		case errors.Is(err, apperrors.ErrInternal):
			message = "internal server error"
		}

		http.Error(w, message, statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(shortenResponse{
		ShortCode: shortCode,
	}); err != nil {
		return
	}

}

func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	shortCode := r.PathValue("short_code")

	longURL, err := h.service.GetLongURL(r.Context(), shortCode)
	if err != nil {
		http.Error(
			w,
			"not found",
			apperrors.GetStatusCode(err),
		)
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, apiKey, err := h.service.CreateUser(r.Context(), req.Email)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidEmail):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	response := createUserResponse{
		UserID: user.ID,
		Email:  user.Email,
		APIKey: apiKey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

}
