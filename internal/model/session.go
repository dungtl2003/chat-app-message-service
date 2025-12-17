package model

import "dungtl2003/chat-app-message-service/internal/types"

type Session struct {
	Id               types.JsonInt64    `json:"id"`
	Version          types.JsonInt64    `json:"version"`
	DeviceInfo       types.Json         `json:"device_info"`
	RefreshTokenHash string             `json:"refresh_token_hash"`
	RevokedAt        types.JsonNullTime `json:"revoked_at"`
	IssuedAt         types.JsonTime     `json:"issued_at"`
	ExpiresAt        types.JsonTime     `json:"expires_at"`
	UserId           types.JsonInt64    `json:"user_id"`
}
