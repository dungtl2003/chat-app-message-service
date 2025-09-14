package kafka

import (
	"dungtl2003/chat-app-message-service/internal/model"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	consumerGroupID                      = "chat-app-message-service-consumer-group"
	MESSAGE_RESOURCE_CREATED_TOPIC Topic = "message-resource-created"
)

type Topic string

type DLQEvent struct {
	SourceTopic   Topic     `json:"source_topic"`
	OriginalKey   []byte    `json:"original_key"`
	OriginalValue []byte    `json:"original_value"`
	ErrorMessage  string    `json:"error_message"`
	Timestamp     time.Time `json:"timestamp"`
}

func (e DLQEvent) ToJson() []byte {
	data, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return data
}

type MessageResourceCreatedEvent struct {
	Message model.Message `json:"message"`
}

func (e MessageResourceCreatedEvent) ToJson() []byte {
	data, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return data
}

type KEvent interface {
	DLQEvent | MessageResourceCreatedEvent
	ToJson() []byte
}

type KMessage[T KEvent] struct {
	Topic Topic  `json:"topic"`
	Key   string `json:"key"`
	Value T      `json:"value"`
}

func (m KMessage[T]) ToKafkaMessage() kafka.Message {
	return kafka.Message{
		Topic: string(m.Topic),
		Key:   []byte(m.Key),
		Value: m.Value.ToJson(),
	}
}
