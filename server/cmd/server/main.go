package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/config"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/handler"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/idgen"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/storage"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config: ", err)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, config.DatabaseURL)
	if err != nil {
		log.Fatal("failed to create database pool: ", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("failed to ping database: ", err)
	}

	store := storage.NewPostgresStore(pool)

	idGen, err := idgen.NewGenerator(config.NodeID)
	if err != nil {
		log.Fatal("failed to create id generator: ", err)
		return
	}

	h := handler.NewHandler(store, idGen)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/shorten", h.HandleShorten)
	mux.HandleFunc("GET /{short_code}", h.HandleRedirect)

	addr := ":" + config.ServerPort
	log.Println("starting server on port", config.ServerPort)

	log.Fatal(http.ListenAndServe(addr, mux))

}
