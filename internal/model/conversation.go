package model

import (
	"dungtl2003/chat-app-message-service/internal/types"
)

type ConversationType string

const (
	PRIVATE ConversationType = "PRIVATE"
	GROUP   ConversationType = "GROUP"
)

type Conversation struct {
	Id        types.JsonInt64    `json:"id"`
	Type      ConversationType   `json:"type"`
	CreatedAt types.JsonTime     `json:"created_at"`
	DeletedAt types.JsonNullTime `json:"deleted_at"`
	CreatorId types.JsonInt64    `json:"creator_id"`
}

func IsConversationType(s string) bool {
	return s == string(PRIVATE) || s == string(GROUP)
}
