package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/apperrors"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/idgen"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage/db"
	"github.com/SHIVAM-KUMAR-59/url-shortener/pkg/base62"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	store storage.URLStore
	idGen *idgen.Generator
}

func NewService(store storage.URLStore, idGen *idgen.Generator) *Service {
	return &Service{
		store: store,
		idGen: idGen,
	}
}

func (s *Service) CreateShortURL(
	ctx context.Context,
	longURL string,
) (string, error) {
	if longURL == "" {
		return "", apperrors.ErrInvalidLongURL
	}

	id, err := s.idGen.NextID()
	if err != nil {
		return "", apperrors.ErrInternal
	}

	shortCode := base62.Encode(id)

	hashBytes := sha256.Sum256([]byte(longURL))
	longURLHash := hex.EncodeToString(hashBytes[:])

	_, err = s.store.CreateURL(ctx, db.CreateURLParams{
		ID:            int64(id),
		ShortCode:     shortCode,
		LongUrl:       longURL,
		LongUrlHash:   longURLHash,
		UserID:        pgtype.Int8{Valid: false},
		IsCustomAlias: false,
		ExpiresAt:     pgtype.Timestamptz{Valid: false},
	})
	if err != nil {
		return "", apperrors.ErrInternal
	}

	return shortCode, nil

}

func (s *Service) GetLongURL(
	ctx context.Context,
	shortCode string,
) (string, error) {
	if shortCode == "" {
		return "", apperrors.ErrURLNotFound
	}

	url, err := s.store.GetURLByShortCode(ctx, shortCode)
	if err != nil {
		return "", apperrors.ErrURLNotFound
	}

	if !url.IsActive {
		return "", apperrors.ErrURLInactive
	}

	if url.ExpiresAt.Valid && url.ExpiresAt.Time.Before(time.Now()) {
		return "", apperrors.ErrURLExpired
	}

	return url.LongUrl, nil

}
