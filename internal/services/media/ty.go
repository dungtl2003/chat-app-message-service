package media

import (
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services"
	"dungtl2003/chat-app-message-service/internal/types"
	"fmt"
)

const (
	UP   = "UP"
	DOWN = "DOWN"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type BadResponseError struct {
	ErrBlock types.ErrorBlock
}

func (e BadResponseError) Error() string {
	return fmt.Sprintf("bad response: %s (status code: %d)", e.ErrBlock.Message, e.ErrBlock.Code)
}

type MediaGetBatchRequest struct {
	AssetIds      []types.JsonInt64
	InternalToken string
}

type MediaGetBatchResponse struct {
	Assets map[string]model.Asset `json:"assets"`
}

type MediaService interface {
	services.Service
	GetAssets(req MediaGetBatchRequest) (*MediaGetBatchResponse, error)
}
