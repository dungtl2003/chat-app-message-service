package kafka

import (
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	ConsumerGroupID                      = "chat-app-message-service-consumer-group"
	MESSAGE_RESOURCE_CREATED_TOPIC Topic = "message-resource-created"
	ASSET_RESOURCE_CONFIRM_TOPIC   Topic = "asset-resource-confirm"
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
	Message             model.Message   `json:"message"`
	ConversationEventId types.JsonInt64 `json:"conversation_event_id"`
	ConversationId      types.JsonInt64 `json:"conversation_id"`
	OutboxId            types.JsonInt64 `json:"outbox_id"`
	IdempotencyKey      string          `json:"idempotency_key"`
}

func (e MessageResourceCreatedEvent) ToJson() []byte {
	data, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return data
}

type AssetResourceConfirmEvent struct {
	AssetId types.JsonInt64 `json:"asset_id"`
}

func (e AssetResourceConfirmEvent) ToJson() []byte {
	data, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return data
}

type KEvent interface {
	DLQEvent | MessageResourceCreatedEvent |
		AssetResourceConfirmEvent

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

func CreateEventKey(conversationId int64) string {
	return fmt.Sprintf("conversation-%d", conversationId)
}
