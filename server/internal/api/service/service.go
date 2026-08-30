package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/analytics"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/apperrors"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/cache"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/events"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/idgen"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage/db"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/utils"
	"github.com/SHIVAM-KUMAR-59/url-shortener/pkg/base62"
	"github.com/jackc/pgx/v5/pgtype"
)

const cacheTTL = time.Hour

type Service struct {
	store           storage.Store
	cacheStore      cache.Cache
	idGen           *idgen.Generator
	publisher       events.Publisher
	analyticsReader *analytics.ClickHouseReader
}

func NewService(
	store storage.Store,
	cacheStore cache.Cache,
	idGen *idgen.Generator,
	publisher events.Publisher,
	analyticsReader *analytics.ClickHouseReader,
) *Service {
	return &Service{
		store:           store,
		cacheStore:      cacheStore,
		idGen:           idGen,
		publisher:       publisher,
		analyticsReader: analyticsReader,
	}
}

func (s *Service) CreateShortURL(ctx context.Context, longURL string, userID *int64) (string, error) {
	if longURL == "" {
		return "", apperrors.ValidationError("invalid URL provided")
	}

	hashBytes := sha256.Sum256([]byte(longURL))
	longURLHash := hex.EncodeToString(hashBytes[:])

	// Idempotent dedup: only applies to authenticated users.
	// If this user already shortened this exact URL, return the
	// existing short_code instead of attempting a new insert.
	if userID != nil {
		existing, err := s.store.GetURLByUserAndHash(ctx, db.GetURLByUserAndHashParams{
			UserID:      pgtype.Int8{Int64: *userID, Valid: true},
			LongUrlHash: longURLHash,
		})
		if err == nil {
			return existing.ShortCode, nil
		}
	}

	id, err := s.idGen.NextID()
	if err != nil {
		log.Printf("failed to generate URL ID: %v", err)
		return "", apperrors.InternalServerError("something went wrong, please try again")
	}

	shortCode := base62.Encode(id)

	_, err = s.store.CreateURL(ctx, db.CreateURLParams{
		ID:            int64(id),
		ShortCode:     shortCode,
		LongUrl:       longURL,
		LongUrlHash:   longURLHash,
		UserID:        pgtype.Int8{Int64: utils.DerefOrZero(userID), Valid: userID != nil},
		IsCustomAlias: false,
		ExpiresAt:     pgtype.Timestamptz{Valid: false},
	})
	if err != nil {
		log.Printf("failed to create URL: %v", err)
		return "", apperrors.InternalServerError("something went wrong, please try again")
	}

	if err := s.cacheStore.Set(ctx, shortCode, longURL, cacheTTL); err != nil {
		log.Printf("failed to cache URL for short code %s: %v", shortCode, err)
	}

	return shortCode, nil
}

func (s *Service) GetLongURL(
	ctx context.Context,
	shortCode string,
	clickEvent events.ClickEvent,
) (string, error) {
	if shortCode == "" {
		return "", apperrors.BadRequestError("invalid code entered")
	}

	// Fast path: return immediately on cache hit.
	longURL, err := s.cacheStore.Get(ctx, shortCode)
	if err == nil {
		// Add to kafka
		go func() {
			publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			clickEvent.ShortCode = shortCode
			clickEvent.ClickedAt = time.Now()
			clickEvent.StatusCode = http.StatusFound

			if err := s.publisher.PublishClickEvent(publishCtx, clickEvent); err != nil {
				log.Printf("failed to publish click event for %s: %v", shortCode, err)
			}
		}()

		return longURL, nil
	}

	// Cache miss and Redis failures both gracefully fall back to PostgreSQL.
	if !errors.Is(err, apperrors.ErrCacheMiss) {
		log.Printf("cache get failed for short code %s: %v", shortCode, err)
	}

	url, err := s.store.GetURLByShortCode(ctx, shortCode)
	if err != nil {
		// Ideally distinguish "no rows" from an actual database failure.
		// For now, avoid exposing storage details to the client.
		log.Printf("failed to get URL for short code %s: %v", shortCode, err)
		return "", apperrors.NotFoundError("URL", shortCode)
	}

	if !url.IsActive {
		return "", apperrors.InactiveError("this URL is inactive")
	}

	if url.ExpiresAt.Valid && url.ExpiresAt.Time.Before(time.Now()) {
		return "", apperrors.ExpiredError("this URL has expired")
	}

	// Successful DB lookup: populate cache.
	if err := s.cacheStore.Set(ctx, shortCode, url.LongUrl, cacheTTL); err != nil {
		log.Printf("failed to cache URL for short code %s: %v", shortCode, err)
	}

	// Add to kafka
	go func() {
		publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		clickEvent.ShortCode = shortCode
		clickEvent.ClickedAt = time.Now()
		clickEvent.StatusCode = http.StatusFound

		if err := s.publisher.PublishClickEvent(publishCtx, clickEvent); err != nil {
			log.Printf("failed to publish click event for %s: %v", shortCode, err)
		}
	}()

	return url.LongUrl, nil

}

func (s *Service) CreateUser(
	ctx context.Context,
	email string,
) (db.User, string, error) {
	normalizedEmail, err := utils.NormalizeEmail(email)
	if err != nil {
		return db.User{}, "", apperrors.ValidationError("invalid email address")
	}

	id, err := s.idGen.NextID()
	if err != nil {
		log.Printf("failed to generate user ID: %v", err)
		return db.User{}, "", apperrors.InternalServerError(
			"something went wrong, please try again",
		)
	}

	apiKey, err := utils.GenerateAPIKey()
	if err != nil {
		log.Printf("failed to generate API key: %v", err)
		return db.User{}, "", apperrors.InternalServerError(
			"something went wrong, please try again",
		)
	}

	hashedAPIKey := utils.HashAPIKey(apiKey)

	user, err := s.store.CreateUser(ctx, db.CreateUserParams{
		ID:         int64(id),
		Email:      normalizedEmail,
		ApiKeyHash: hashedAPIKey,
	})
	if err != nil {
		log.Printf("failed to create user: %v", err)
		return db.User{}, "", apperrors.InternalServerError(
			"something went wrong, please try again",
		)
	}

	return user, apiKey, nil

}

func (s *Service) GetUserByAPIKeyHash(
	ctx context.Context,
	apiKeyHash string,
) (db.User, error) {
	if apiKeyHash == "" {
		return db.User{}, apperrors.UnauthorizedError("invalid API key")
	}

	user, err := s.store.GetUserByAPIKeyHash(ctx, apiKeyHash)
	if err != nil {
		log.Printf("failed to find user by API key hash: %v", err)
		return db.User{}, apperrors.UnauthorizedError("invalid API key")
	}

	return user, nil

}

type StatsResult struct {
	ShortCode   string
	LongURL     string
	CreatedAt   time.Time
	TotalClicks int64
	LastClicked *time.Time
}

func (s *Service) GetURLStats(ctx context.Context, shortCode string, userID int64) (StatsResult, error) {
	url, err := s.store.GetURLByShortCode(ctx, shortCode)

	if err != nil {
		return StatsResult{}, apperrors.NotFoundError("url", shortCode)
	}

	if !url.UserID.Valid || url.UserID.Int64 != userID {
		return StatsResult{}, apperrors.NotFoundError("url", shortCode)
	}

	totalClicks, lastClicked, err := s.analyticsReader.GetStats(ctx, shortCode)
	if err != nil {
		return StatsResult{}, apperrors.InternalServerError("Something went wrong, please try again.")
	}

	var result StatsResult
	result = StatsResult{
		ShortCode:   shortCode,
		LongURL:     url.LongUrl,
		CreatedAt:   url.CreatedAt.Time,
		TotalClicks: totalClicks,
		LastClicked: lastClicked,
	}

	return result, nil
}
