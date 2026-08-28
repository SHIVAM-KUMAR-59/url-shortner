package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/api/handler"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/api/service"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/cache"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/config"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/idgen"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config: ", err)
	}

	ctx := context.Background()

	// PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("failed to create database pool: ", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("failed to ping database: ", err)
	}

	store := storage.NewPostgresStore(pool)

	// Redis
	options, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatal("failed to parse Redis URL: ", err)
	}

	redisClient := redis.NewClient(options)
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("failed to ping Redis: ", err)
	}

	cacheStore := cache.NewRedisCache(redisClient)

	// ID Generator
	idGen, err := idgen.NewGenerator(cfg.NodeID)
	if err != nil {
		log.Fatal("failed to create ID generator: ", err)
	}

	// Application Service
	svc := service.NewService(store, cacheStore, idGen)

	// HTTP Handler
	h := handler.NewHandler(svc)

	// Routes
	mux := http.NewServeMux()

	mux.Handle("POST /api/v1/shorten", h.AuthMiddleware(http.HandlerFunc(h.HandleShorten)))
	mux.HandleFunc("GET /{short_code}", h.HandleRedirect)

	mux.HandleFunc("POST /api/v1/users", h.HandleCreateUser)

	addr := ":" + cfg.ServerPort
	log.Println("starting server on port", cfg.ServerPort)

	log.Fatal(http.ListenAndServe(addr, mux))

}
