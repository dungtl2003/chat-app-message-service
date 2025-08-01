package model

import "dungtl2003/chat-app-message-service/internal/types"

type ParticipantRole string

const (
	LEADER    ParticipantRole = "LEADER"
	MODERATOR ParticipantRole = "MODERATOR"
	MEMBER    ParticipantRole = "MEMBER"
)

type Participant struct {
	Id                types.JsonInt64      `json:"id"`
	UserId            types.JsonInt64      `json:"user_id"`
	Name              string               `json:"name"`
	Avatar            types.JsonNullString `json:"avatar"`
	Role              ParticipantRole      `json:"role"`
	ConversationId    types.JsonInt64      `json:"conversation_id"`
	JoinedAt          types.JsonTime       `json:"joined_at"`
	LeftAt            types.JsonNullTime   `json:"left_at"`
	ClearedMessagesAt types.JsonNullTime   `json:"cleared_messages_at"`
}
