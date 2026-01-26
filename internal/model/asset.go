package model

import (
	"dungtl2003/chat-app-message-service/internal/types"
)

type AssetKind string
type AssetStatus string

const (
	AssetKindImage AssetKind = "IMAGE"
	AssetKindVideo AssetKind = "VIDEO"
	AssetKindAudio AssetKind = "AUDIO"
	AssetKindFile  AssetKind = "FILE"
	AssetKindOther AssetKind = "OTHER"

	AssetStatusPending  AssetStatus = "PENDING"
	AssetStatusUploaded AssetStatus = "UPLOADED"
	AssetStatusReady    AssetStatus = "READY"
	AssetStatusBlocked  AssetStatus = "BLOCKED"
)

var (
	AllowedAssetKinds = map[AssetKind]bool{
		AssetKindImage: true,
		AssetKindVideo: true,
		AssetKindAudio: true,
		AssetKindFile:  true,
		AssetKindOther: true,
	}

	AllowedAssetStatuses = map[AssetStatus]bool{
		AssetStatusPending:  true,
		AssetStatusUploaded: true,
		AssetStatusReady:    true,
		AssetStatusBlocked:  true,
	}
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
	OriginalFilename string               `json:"original_filename"`
	ApiKey           types.JsonNullString `json:"api_key"`
	Bucket           types.JsonNullString `json:"bucket"`
	ObjectKey        types.JsonNullString `json:"object_key"`
	Mime             types.JsonNullString `json:"mime"`
	Size             types.JsonNullInt64  `json:"size"`
	Sha256Hash       types.JsonNullString `json:"sha256_hash"`
	Kind             AssetKind            `json:"kind"`
	AssetStatus      AssetStatus          `json:"asset_status"`
	OwnerUserId      types.JsonNullInt64  `json:"owner_user_id"`
}

func (a *Asset) DeepCopy() *Asset {
	if a == nil {
		return nil
	}

	return &Asset{
		Id:               a.Id,
		PublicId:         a.PublicId,
		Width:            a.Width,
		Height:           a.Height,
		Format:           a.Format,
		ResourceType:     a.ResourceType,
		CreatedAt:        a.CreatedAt,
		Bytes:            a.Bytes,
		Url:              a.Url,
		SecureUrl:        a.SecureUrl,
		AssetFolder:      a.AssetFolder,
		OriginalFilename: a.OriginalFilename,
		ApiKey:           a.ApiKey,
		Bucket:           a.Bucket,
		ObjectKey:        a.ObjectKey,
		Mime:             a.Mime,
		Size:             a.Size,
		Sha256Hash:       a.Sha256Hash,
		Kind:             a.Kind,
		AssetStatus:      a.AssetStatus,
		OwnerUserId:      a.OwnerUserId,
	}
}
