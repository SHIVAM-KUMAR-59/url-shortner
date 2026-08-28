package storage

import (
	"context"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage/db"
)

type UserStore interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	GetUserByAPIKeyHash(ctx context.Context, apiHashKey string) (db.User, error)
}

type URLStore interface {
	CreateURL(ctx context.Context, arg db.CreateURLParams) (db.Url, error)
	GetURLByShortCode(ctx context.Context, shortCode string) (db.Url, error)
	GetURLByUserAndHash(ctx context.Context, arg db.GetURLByUserAndHashParams) (db.Url, error)
}

type Store interface {
	URLStore
	UserStore
}
