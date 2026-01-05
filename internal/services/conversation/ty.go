package conversation

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

type BatchGetParticipantsRequest struct {
	ConversationID int64   `json:"conversation_id"`
	UserIDs        []int64 `json:"user_ids"`
	InternalToken  string  `json:"internal_token"`
}

type BatchGetParticipantsResponse struct {
	ParticipantMap map[int64]model.Participant `json:"participant_map"`
}

type ConversationService interface {
	services.Service
	BatchGetParticipants(req *BatchGetParticipantsRequest) (*BatchGetParticipantsResponse, error)
}
