package storage

import (
	"context"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage/db"
)

type URLStore interface {
	CreateURL(ctx context.Context, arg db.CreateURLParams) (db.Url, error)
	GetURLByShortCode(ctx context.Context, shortCode string) (db.Url, error)
	GetURLByUserAndHash(ctx context.Context, arg db.GetURLByUserAndHashParams) (db.Url, error)
}
