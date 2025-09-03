package media

import (
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services"
	"mime/multipart"
)

const (
	FORM_FIELD_ASSET = "asset"
	UP               = "UP"
	DOWN             = "DOWN"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type AssetBatchPostResponseBody struct {
	Assets []model.Asset `json:"assets"`
	Error  string        `json:"error"`
}

type AssetBatchPostParams struct {
	AssetIds []int64 `json:"asset_ids"`
}

type AssetBatchPostResponse struct {
	StatusCode int                        `json:"status_code"`
	Body       AssetBatchPostResponseBody `json:"body"`
}

type AssetGetParams struct {
	AssetId int64 `json:"asset_id"`
}

type AssetGetResponseBody struct {
	Asset model.Asset `json:"asset"`
	Error string      `json:"error"`
}

type AssetGetResponse struct {
	StatusCode int                  `json:"status_code"`
	Body       AssetGetResponseBody `json:"body"`
}

type MediaService interface {
	services.Service
	PostAsset(fileHeader multipart.FileHeader, token string) (*AssetPostResponse, error)
	GetAssetBatch(params AssetBatchPostParams) (*AssetBatchPostResponse, error)
	GetAsset(params AssetGetParams) (*AssetGetResponse, error)
}

type AssetPostResponseBody struct {
	Asset model.Asset `json:"asset"`
	Error string      `json:"error"`
}

type AssetPostResponse struct {
	StatusCode int                   `json:"status_code"`
	Body       AssetPostResponseBody `json:"body"`
}
