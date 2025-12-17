package model

import "dungtl2003/chat-app-message-service/internal/types"

type GroupChat struct {
	ConversationId types.JsonInt64     `json:"conversation_id"`
	Name           string              `json:"name"`
	AvatarId       types.JsonNullInt64 `json:"avatar_id"`

	AvatarURL types.JsonNullString `json:"avatar_url"`
}
