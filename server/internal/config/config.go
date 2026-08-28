package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	ServerPort  string
	NodeID      int64
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

	nodeIDStr := os.Getenv("NODE_ID")
	if nodeIDStr == "" {
		nodeIDStr = "1"
	}

	nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
	if err != nil {
		return nil, errors.New("NODE_ID must be a valid integer")
	}

	return &Config{
		DatabaseURL: databaseURL,
		ServerPort:  serverPort,
		NodeID:      nodeID,
	}, nil

}
