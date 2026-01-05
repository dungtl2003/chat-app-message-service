package user

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

type GetUsersRequest struct {
	UserIDs       []int64 `json:"user_ids"`
	InternalToken string  `json:"internal_token"`
}

type GetUsersResponse struct {
	UserMap map[int64]model.ChatUser `json:"user_map"`
}

type UserService interface {
	services.Service
	GetUsers(req *GetUsersRequest) (*GetUsersResponse, error)
}
