package kafka

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type KafkaWriterService struct {
	status  services.ServiceStatus
	w       *kafka.Writer
	ctx     context.Context
	logger  *logging.LoggerWrapper
	dlqChan <-chan KMessage[DLQEvent] // channel to receive messages for DLQ
}

// New creates a new Kafka writer that can be used to write messages.
// Remember to call Close() when done to release resources.
func NewKafkaWriterService(brokers []string, logger *logging.LoggerWrapper, ctx context.Context, dlqChan <-chan KMessage[DLQEvent]) (*KafkaWriterService, error) {
	if len(brokers) < 1 {
		return nil, fmt.Errorf("producer needs atleast 1 broker")
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.Hash{},
		MaxAttempts:  10,
		RequiredAcks: kafka.RequireOne,
	}

	k := &KafkaWriterService{
		w:       w,
		ctx:     ctx,
		logger:  logger,
		status:  services.READY,
		dlqChan: dlqChan,
	}

	go func() {
		for msg := range k.dlqChan {
			if err := WriteMessages(k, []KMessage[DLQEvent]{msg}); err != nil {
				k.logger.Errorfln("[%s] Failed to write message to DLQ: %v", k.Name(), err)
				k.status = services.ERROR
				continue
			}
			k.logger.Debugfln("[%s] Message written to DLQ: %s", k.Name(), msg.Key)
		}
	}()

	logger.Infofln("[%s] Kafka writer created with brokers: %v", k.Name(), brokers)
	logger.Infofln("[%s] Running", k.Name())
	return k, nil
}

func (k *KafkaWriterService) Status() services.ServiceStatus {
	return k.status
}

func (k *KafkaWriterService) Name() string {
	return "Kafka Writer Service"
}

// WriteMessages writes a message to the specified topic with the given key and value.
// The value is expected to be a struct that implements the Event interface.
// Because Go does not support generic methods, we make a normal function here.
func WriteMessages[T KEvent](k *KafkaWriterService, messages []KMessage[T]) error {
	if len(messages) == 0 {
		return nil
	}

	kmessages := make([]kafka.Message, len(messages))
	for i, msg := range messages {
		topic := string(msg.Topic)
		key := []byte(msg.Key)
		value, err := json.Marshal(msg.Value)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		m := kafka.Message{
			Topic: topic,
			Key:   key,
			Value: value,
		}

		k.logger.Debugfln("[%s] Writing message to topic %s with key %s, payload: %s", k.Name(), m.Topic, m.Key, m.Value)
		kmessages[i] = m
	}

	if err := k.w.WriteMessages(k.ctx, kmessages...); err != nil {
		k.status = services.ERROR
		return fmt.Errorf("failed to write messages: %w", err)
	}

	return nil
}

// Close close the writer's connection. Note that this needs to be called when
// the process exits.
func (k *KafkaWriterService) Close() error {
	if k.status == services.STOPPED {
		k.logger.Errorfln("[%s] Already stopped", k.Name())
		return nil
	}

	k.logger.Infofln("[%s] Closing", k.Name())
	if err := k.w.Close(); err != nil {
		k.logger.Errorfln("[%s] Failed to close writer, error: %v", k.Name(), err)
		k.status = services.ERROR
		return err
	}

	k.logger.Infofln("[%s] Writer closed successfully", k.Name())
	k.status = services.STOPPED
	return nil
}
