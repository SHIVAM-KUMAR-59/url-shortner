package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/apperrors"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/cache"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/idgen"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage/db"
	"github.com/SHIVAM-KUMAR-59/url-shortener/pkg/base62"
	"github.com/jackc/pgx/v5/pgtype"
)

const cacheTTL = time.Hour

type Service struct {
	store      storage.URLStore
	cacheStore cache.Cache
	idGen      *idgen.Generator
}

func NewService(
	store storage.URLStore,
	cacheStore cache.Cache,
	idGen *idgen.Generator,
) *Service {
	return &Service{
		store:      store,
		cacheStore: cacheStore,
		idGen:      idGen,
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

	if err := s.cacheStore.Set(ctx, shortCode, longURL, cacheTTL); err != nil {
		log.Printf("failed to cache URL for short code %s: %v", shortCode, err)
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

	// Fast path: return immediately on cache hit.
	longURL, err := s.cacheStore.Get(ctx, shortCode)
	if err == nil {
		return longURL, nil
	}

	// Graceful degradation:
	// - Cache miss: fall through to DB.
	// - Redis/cache failure: also fall through to DB.
	if err != nil && !errors.Is(err, apperrors.ErrCacheMiss) {
		log.Printf("cache get failed for short code %s: %v", shortCode, err)
	}

	// Cache miss or cache failure: fetch from the database.
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

	// Populate cache after a successful DB lookup.
	// Cache failures should not break a successful redirect.
	if err := s.cacheStore.Set(ctx, shortCode, url.LongUrl, cacheTTL); err != nil {
		log.Printf("failed to cache URL for short code %s: %v", shortCode, err)
	}

	return url.LongUrl, nil

}
