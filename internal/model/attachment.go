package model

import "dungtl2003/chat-app-message-service/internal/types"

type Attachment struct {
	Id        types.JsonInt64    `json:"id"`
	ThumbURL  string             `json:"thumb_url"`
	FileURL   string             `json:"file_url"`
	DeletedAt types.JsonNullTime `json:"deleted_at"`
	MessageId types.JsonInt64    `json:"message_id"`
}
