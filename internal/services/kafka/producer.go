package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
	logger *logging.LoggerWrapper
}

// NewKafkaProducer creates the wrapper. No channels passed here.
func NewKafkaProducer(brokers []string, logger *logging.LoggerWrapper) (*KafkaProducer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("producer needs at least 1 broker")
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.Hash{}, // Key-based ordering
		MaxAttempts:  10,
		RequiredAcks: kafka.RequireOne,
		BatchSize:    100, // Optimize throughput
		BatchTimeout: 10 * time.Millisecond,
		// Async:    true, // Optional: writes return immediately, errors reported async
	}

	return &KafkaProducer{
		writer: w,
		logger: logger,
	}, nil
}

// Write publishes a generic message.
// We accept any for value and marshal it internally for convenience.
func (p *KafkaProducer) Write(ctx context.Context, topic string, key string, value any) error {
	// Marshal Payload
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal kafka payload: %w", err)
	}

	// Construct Message
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: jsonBytes,
	}

	// Write with Context (Passed from caller, not stored in struct)
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.logger.Errorfln("[%s] Failed to write to kafka topic %s: %v", p.Name(), topic, err)
		return err
	}

	p.logger.Debugfln("[%s] Successfully wrote to %s (key: %s)", p.Name(), topic, key)
	return nil
}

func (p *KafkaProducer) Close() error {
	p.logger.Infofln("Closing Kafka Producer")
	return p.writer.Close()
}

func (p *KafkaProducer) Name() string {
	return "Kafka Producer"
}

func (p *KafkaProducer) Status() services.ServiceStatus {
	// Note: kafka-go writer does not expose status, so we assume ready if not closed.
	return services.ServiceReady
}
