package config

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	ServerPort    string
	RedisURL      string
	EtcdEndpoints string
	KafkaBrokers  []string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using environment variables")
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
		redisURL = "redis://redis:6379/0"
	}

	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		etcdEndpoints = "etcd:2379"
	}

	var kafkaBrokers []string
	kafkaEnvBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaEnvBrokers == "" {
		kafkaBrokers = []string{"kafka:9092"}
	} else {
		kafkaBrokers = strings.Split(kafkaEnvBrokers, ",")
	}

	return &Config{
		DatabaseURL:   databaseURL,
		ServerPort:    serverPort,
		RedisURL:      redisURL,
		EtcdEndpoints: etcdEndpoints,
		KafkaBrokers:  kafkaBrokers,
	}, nil
}
