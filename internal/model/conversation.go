package model

import (
	"dungtl2003/chat-app-message-service/internal/types"
)

type ConversationType string

const (
	DIRECT ConversationType = "DIRECT"
	GROUP  ConversationType = "GROUP"
)

type Conversation struct {
	Id        types.JsonInt64    `json:"id"`
	Type      ConversationType   `json:"type"`
	CreatedAt types.JsonTime     `json:"created_at"`
	DeletedAt types.JsonNullTime `json:"deleted_at"`
	UpdatedAt types.JsonTime     `json:"updated_at"`

	LastActivityAt types.JsonNullTime  `json:"last_activity_at"`
	LastMessageId  types.JsonNullInt64 `json:"last_message_id"`

	Group        *GroupChat    `json:"group"`
	Participants []Participant `json:"participants"`
}
