package model

import "dungtl2003/chat-app-message-service/internal/types"

type GroupChat struct {
	Id             types.JsonInt64      `json:"id"`
	Name           string               `json:"name"`
	Avatar         types.JsonNullString `json:"avatar"`
	UpdatedAt      types.JsonNullTime   `json:"updated_at"`
	ConversationId types.JsonInt64      `json:"conversation_id"`
}
