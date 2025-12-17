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
	Id            types.JsonInt64     `json:"id"`
	Type          ConversationType    `json:"type"`
	CreatedAt     types.JsonTime      `json:"created_at"`
	UpdatedAt     types.JsonTime      `json:"updated_at"`
	DeletedAt     types.JsonNullTime  `json:"deleted_at"`
	Version       types.JsonInt64     `json:"version"`
	LatestEventId types.JsonNullInt64 `json:"latest_event_id"`

	Group        *GroupChat    `json:"group"`
	Participants []Participant `json:"participants"`
}
