package storage

import (
	"context"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	queries *db.Queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		queries: db.New(pool),
	}
}

func (s *PostgresStore) CreateURL(
	ctx context.Context,
	arg db.CreateURLParams,
) (db.Url, error) {
	return s.queries.CreateURL(ctx, arg)
}

func (s *PostgresStore) GetURLByShortCode(
	ctx context.Context,
	shortCode string,
) (db.Url, error) {
	return s.queries.GetURLByShortCode(ctx, shortCode)
}

func (s *PostgresStore) GetURLByUserAndHash(
	ctx context.Context,
	arg db.GetURLByUserAndHashParams,
) (db.Url, error) {
	return s.queries.GetURLByUserAndHash(ctx, arg)
}
