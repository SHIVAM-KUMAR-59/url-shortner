package events

import (
	"context"
	"encoding/json"

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

func (p *KafkaPublisher) PublishClickEvent(ctx context.Context, event ClickEvent) error {

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(event.ShortCode),
		Value: payload,
	}

	return p.kafkaWriter.WriteMessages(ctx, msg)
}

func (p *KafkaPublisher) Close() {
	p.kafkaWriter.Close()
}
