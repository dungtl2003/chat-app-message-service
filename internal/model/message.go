package model

import (
	"dungtl2003/chat-app-message-service/internal/types"
)

type MessageType string

const (
	TEXT     = "TEXT"
	IMAGE    = "IMAGE"
	VIDEO    = "VIDEO"
	AUDIO    = "AUDIO"
	FILE     = "FILE"
	GIF      = "GIF"
	STICKER  = "STICKER"
	LOCATION = "LOCATION"
	POLL     = "POLL"
)

type Message struct {
	Id         types.JsonInt64    `json:"id"`
	Content    string             `json:"content"`
	Type       MessageType        `json:"type"`
	CreatedAt  types.JsonTime     `json:"created_at"`
	UpdatedAt  types.JsonNullTime `json:"updated_at"`
	DeletedAt  types.JsonNullTime `json:"deleted_at"`
	SenderId   types.JsonInt64    `json:"sender_id"`
	ReceiverId types.JsonInt64    `json:"receiver_id"`

	Attachments []Attachment `json:"attachments"`
}

func IsMessageType(s string) bool {
	return s == TEXT || s == IMAGE || s == VIDEO || s == AUDIO || s == FILE || s == GIF || s == STICKER || s == LOCATION || s == POLL
}
