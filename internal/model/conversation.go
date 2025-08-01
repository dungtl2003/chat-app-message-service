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

	Group        *GroupChat    `json:"group"`
	Participants []Participant `json:"participants"`
}

func IsConversationType(s string) bool {
	return s == string(DIRECT) || s == string(GROUP)
}
