package model

import "dungtl2003/chat-app-message-service/internal/types"

type GroupChat struct {
	Id             types.JsonInt64     `json:"id"`
	Name           string              `json:"name"`
	AvatarId       types.JsonNullInt64 `json:"avatar_id"`
	ConversationId types.JsonInt64     `json:"conversation_id"`
}
