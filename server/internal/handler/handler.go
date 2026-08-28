package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/idgen"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage/db"
	"github.com/SHIVAM-KUMAR-59/url-shortener/pkg/base62"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	store storage.URLStore
	idGen *idgen.Generator
}

func NewHandler(store storage.URLStore, idGen *idgen.Generator) *Handler {
	return &Handler{
		store: store,
		idGen: idGen,
	}
}

func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	shortCode := r.PathValue("short_code")

	if shortCode == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	url, err := h.store.GetURLByShortCode(r.Context(), shortCode)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if !url.IsActive {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if url.ExpiresAt.Valid && url.ExpiresAt.Time.Before(time.Now()) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, url.LongUrl, http.StatusFound)
}

type shortenRequest struct {
	LongURL string `json:"long_url"`
}

type shortenResponse struct {
	ShortCode string `json:"short_code"`
}

func (h *Handler) HandleShorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest

	// 1. Decode request body.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// 2. Validate long URL.
	if req.LongURL == "" {
		http.Error(w, "long_url is required", http.StatusBadRequest)
		return
	}

	// 3. Generate a unique ID.
	id, err := h.idGen.NextID()
	if err != nil {
		http.Error(w, "failed to create short URL", http.StatusInternalServerError)
		return
	}

	// 4. Convert the ID into a Base62 short code.
	shortCode := base62.Encode(id)

	// 5. Create a SHA-256 hash of the long URL.
	hashBytes := sha256.Sum256([]byte(req.LongURL))
	longURLHash := hex.EncodeToString(hashBytes[:])

	// 6. Create the URL record.
	_, err = h.store.CreateURL(r.Context(), db.CreateURLParams{
		ID:            int64(id),
		ShortCode:     shortCode,
		LongUrl:       req.LongURL,
		LongUrlHash:   longURLHash,
		UserID:        pgtype.Int8{Valid: false},
		IsCustomAlias: false,
		ExpiresAt:     pgtype.Timestamptz{Valid: false},
	})
	if err != nil {
		http.Error(w, "failed to create short URL", http.StatusInternalServerError)
		return
	}

	// 7. Return JSON response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(shortenResponse{
		ShortCode: shortCode,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

}
