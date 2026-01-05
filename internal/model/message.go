package model

import (
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
)

type MessageType string
type OutboxStatus string

const (
	MSG_TEXT     MessageType = "TEXT"
	MSG_IMAGE    MessageType = "IMAGE"
	MSG_VIDEO    MessageType = "VIDEO"
	MSG_AUDIO    MessageType = "AUDIO"
	MSG_FILE     MessageType = "FILE"
	MSG_GIF      MessageType = "GIF"
	MSG_STICKER  MessageType = "STICKER"
	MSG_LOCATION MessageType = "LOCATION"
	MSG_POLL     MessageType = "POLL"
	MSG_SYSTEM   MessageType = "SYSTEM"

	OUTBOX_PENDING OutboxStatus = "PENDING"
	OUTBOX_SENT    OutboxStatus = "SENT"
)

type Message struct {
	Id               types.JsonInt64     `json:"id"`
	Content          string              `json:"content"`
	Type             MessageType         `json:"type"`
	CreatedAt        types.JsonTime      `json:"created_at"`
	UpdatedAt        types.JsonTime      `json:"updated_at"`
	DeletedAt        types.JsonNullTime  `json:"deleted_at"`
	SenderId         types.JsonInt64     `json:"sender_id"`
	ReceiverId       types.JsonInt64     `json:"receiver_id"`
	ReplyToMessageId types.JsonNullInt64 `json:"reply_to_message_id"`
	Version          types.JsonInt64     `json:"version"`

	Attachments    []Attachment `json:"attachments"`
	IdempotencyKey string       `json:"idempotency_key"` // for deduplication at producer side (fast path)
}

type MessageOutbox struct {
	Id                  types.JsonInt64    `json:"id"`
	MessageId           types.JsonInt64    `json:"message_id"`
	ConversationId      types.JsonInt64    `json:"conversation_id"`
	ConversationEventId types.JsonInt64    `json:"conversation_event_id"`
	Payload             types.Json         `json:"payload"`
	Status              OutboxStatus       `json:"status"`
	CreatedAt           types.JsonTime     `json:"created_at"`
	ProcessedAt         types.JsonNullTime `json:"processed_at"`
	RetryCount          types.JsonInt64    `json:"retry_count"`
	LastError           string             `json:"last_error"`
	NextRetryAt         types.JsonTime     `json:"next_retry_at"`
}

type MessageOutboxPayload struct {
	Message        Message `json:"message"`
	IdempotencyKey string  `json:"idempotency_key"` // for deduplication at consumer side (slow path)
}

func (m Message) String() string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}
