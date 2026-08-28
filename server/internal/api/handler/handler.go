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

	shortCode, err := h.service.CreateShortURL(r.Context(), req.LongURL)
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
