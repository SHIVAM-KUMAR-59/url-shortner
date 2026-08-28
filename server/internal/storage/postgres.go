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

func (s *PostgresStore) CreateURL(ctx context.Context, arg db.CreateURLParams) (db.Url, error) {
	return s.queries.CreateURL(ctx, arg)
}

func (s *PostgresStore) GetURLByShortCode(ctx context.Context, shortCode string) (db.Url, error) {
	return s.queries.GetURLByShortCode(ctx, shortCode)
}

func (s *PostgresStore) GetURLByUserAndHash(ctx context.Context, arg db.GetURLByUserAndHashParams) (db.Url, error) {
	return s.queries.GetURLByUserAndHash(ctx, arg)
}

func (s *PostgresStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	return s.queries.CreateUser(ctx, arg)
}

func (s *PostgresStore) GetUserByAPIKeyHash(ctx context.Context, apiHashKey string) (db.User, error) {
	return s.queries.GetUserByAPIKeyHash(ctx, apiHashKey)
}
