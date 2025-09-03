package model

import (
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
)

type MessageType string

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

	Attachments []Attachment `json:"attachments"`
}

func (m Message) String() string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}
