package model

import (
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
)

type Asset struct {
	Id               types.JsonInt64      `json:"id"`
	PublicId         types.JsonNullString `json:"public_id"`
	Width            types.JsonNullInt64  `json:"width"`
	Height           types.JsonNullInt64  `json:"height"`
	Format           types.JsonNullString `json:"format"`
	ResourceType     types.JsonNullString `json:"resource_type"`
	CreatedAt        types.JsonTime       `json:"created_at"`
	Bytes            types.JsonNullInt64  `json:"bytes"`
	Url              types.JsonNullString `json:"url"`
	SecureUrl        types.JsonNullString `json:"secure_url"`
	AssetFolder      types.JsonNullString `json:"asset_folder"`
	OriginalFilename types.JsonNullString `json:"original_filename"`
	ApiKey           types.JsonNullString `json:"api_key"`
}

func (a *Asset) String() string {
	b, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(b)
}
