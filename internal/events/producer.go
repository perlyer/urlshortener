package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

const Topic = "clicks"

type ClickEvent struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
	Language  string    `json:"language"`
	IP        string    `json:"ip"`
}

type Producer struct{ writer *kafka.Writer }

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{writer: &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: topic, Balancer: &kafka.LeastBytes{}}}
}

func (p *Producer) Publish(ctx context.Context, e ClickEvent) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("events: marshal: %w", err)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Value: data})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
