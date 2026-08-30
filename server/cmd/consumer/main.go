package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/config"
	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/events"
	"github.com/segmentio/kafka-go"
)

const (
	batchSize     = 1000
	flushInterval = 5 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config: ", err)
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.ClickHouseAddr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: cfg.ClickHouseUser,
			Password: cfg.ClickHousePassword,
		},
	})
	if err != nil {
		log.Fatal("failed to connect to clickhouse: ", err)
	}
	defer conn.Close()

	if err := conn.Ping(context.Background()); err != nil {
		log.Fatal("failed to ping clickhouse: ", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.KafkaTopic,
		GroupID: "clickhouse-consumer",
	})
	defer reader.Close()

	msgChan := make(chan events.ClickEvent)

	// Continuously read from Kafka, push decoded events onto msgChan.
	go func() {
		for {
			msg, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("error reading message: %v", err)
				continue
			}

			var event events.ClickEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("error unmarshaling event: %v", err)
				continue
			}

			log.Printf("received event: %+v", event)

			msgChan <- event
		}
	}()

	batch := make([]events.ClickEvent, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	log.Println("consumer started, waiting for events...")

	for {
		select {
		case event := <-msgChan:
			batch = append(batch, event)
			if len(batch) >= batchSize {
				flush(conn, batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				flush(conn, batch)
				batch = batch[:0]
			}
		}
	}
}

func flush(conn driver.Conn, batch []events.ClickEvent) {
	ctx := context.Background()

	batchInsert, err := conn.PrepareBatch(ctx, "INSERT INTO click_events (short_code, clicked_at)")
	if err != nil {
		log.Printf("failed to prepare batch: %v", err)
		return
	}

	for _, event := range batch {
		if err := batchInsert.Append(event.ShortCode, event.ClickedAt); err != nil {
			log.Printf("failed to append to batch: %v", err)
			continue
		}
	}

	if err := batchInsert.Send(); err != nil {
		log.Printf("failed to send batch: %v", err)
		return
	}

	log.Printf("flushed batch of %d events to clickhouse", len(batch))
}
