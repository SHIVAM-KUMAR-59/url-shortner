package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	kafkaWriter *kafka.Writer
}

func NewKafkaPublisher(brokers []string, topic string) *KafkaPublisher {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &KafkaPublisher{
		kafkaWriter: writer,
	}
}

func (p *KafkaPublisher) PublishClickEvent(ctx context.Context, shortCode string) error {
	event := ClickEvent{
		ShortCode: shortCode,
		ClickedAt: time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(shortCode),
		Value: payload,
	}

	return p.kafkaWriter.WriteMessages(ctx, msg)
}

func (p *KafkaPublisher) Close() {
	p.kafkaWriter.Close()
}
