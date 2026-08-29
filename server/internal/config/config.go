package config

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL        string
	ServerPort         string
	RedisURL           string
	EtcdEndpoints      string
	KafkaBrokers       []string
	KafkaTopic         string
	ClickHouseAddr     string
	ClickHouseUser     string
	ClickHousePassword string
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

	kafkaTopic := "url-events"

	clickHouseAddr := os.Getenv("CLICKHOUSE_ADDR")
	if clickHouseAddr == "" {
		clickHouseAddr = "clickhouse:9000"
	}

	clickHouseUser := os.Getenv("CLICKHOUSE_USER")
	if clickHouseUser == "" {
		return nil, errors.New("CLICKHOUSE_USER is required")
	}

	clickHousePassword := os.Getenv("CLICKHOUSE_PASSWORD")
	if clickHousePassword == "" {
		return nil, errors.New("CLICKHOUSE_PASSWORD is required")
	}

	return &Config{
		DatabaseURL:        databaseURL,
		ServerPort:         serverPort,
		RedisURL:           redisURL,
		EtcdEndpoints:      etcdEndpoints,
		KafkaBrokers:       kafkaBrokers,
		KafkaTopic:         kafkaTopic,
		ClickHouseAddr:     clickHouseAddr,
		ClickHouseUser:     clickHouseUser,
		ClickHousePassword: clickHousePassword,
	}, nil
}
