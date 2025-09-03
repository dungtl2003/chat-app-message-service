package model

import (
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
)

type AttachmentType string

const (
	ATT_IMAGE   AttachmentType = "IMAGE"
	ATT_VIDEO   AttachmentType = "VIDEO"
	ATT_AUDIO   AttachmentType = "AUDIO"
	ATT_FILE    AttachmentType = "FILE"
	ATT_GIF     AttachmentType = "GIF"
	ATT_STICKER AttachmentType = "STICKER"
)

type Attachment struct {
	Id        types.JsonInt64    `json:"id"`
	AssetId   types.JsonInt64    `json:"asset_id"`
	DeletedAt types.JsonNullTime `json:"deleted_at"`
	MessageId types.JsonInt64    `json:"message_id"`
	Position  int                `json:"position"`
	Type      AttachmentType     `json:"type"`
}

func (a Attachment) String() string {
	b, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(b)
}
