package conversation

import (
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services"
)

type MockConversationService struct {
	MockNameFunc             func() string
	MockStatusFunc           func() services.ServiceStatus
	MockCloseFunc            func() error
	MockBatchGetParticipants func(req *BatchGetParticipantsRequest) (*BatchGetParticipantsResponse, error)
	MockIsParticipantFunc    func(conversationID, participantID int64, internalToken string) (bool, error)
}

func (m *MockConversationService) Name() string {
	if m.MockNameFunc != nil {
		return m.MockNameFunc()
	}
	return "Mock Conversation Service"
}

func (m *MockConversationService) Status() services.ServiceStatus {
	if m.MockStatusFunc != nil {
		return m.MockStatusFunc()
	}
	return services.ServiceReady
}

func (m *MockConversationService) Close() error {
	if m.MockCloseFunc != nil {
		return m.MockCloseFunc()
	}
	return nil
}

func (m *MockConversationService) BatchGetParticipants(req *BatchGetParticipantsRequest) (*BatchGetParticipantsResponse, error) {
	if m.MockBatchGetParticipants != nil {
		return m.MockBatchGetParticipants(req)
	}
	return &BatchGetParticipantsResponse{
		ParticipantMap: map[int64]model.Participant{},
	}, nil
}

func (m *MockConversationService) IsParticipant(conversationID, participantID int64, internalToken string) (bool, error) {
	if m.MockIsParticipantFunc != nil {
		return m.MockIsParticipantFunc(conversationID, participantID, internalToken)
	}
	return false, nil
}
