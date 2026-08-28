package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	ServerPort    string
	RedisURL      string
	EtcdEndpoints string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, errors.New("failed to load .env file")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		etcdEndpoints = "http://localhost:2379"
	}

	return &Config{
		DatabaseURL:   databaseURL,
		ServerPort:    serverPort,
		RedisURL:      redisURL,
		EtcdEndpoints: etcdEndpoints,
	}, nil

}
