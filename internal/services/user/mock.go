package user

import (
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services"
)

type MockUserService struct {
	MockNameFunc      func() string
	MockStatusFunc    func() services.ServiceStatus
	MockCloseFunc     func() error
	MockGetUsers      func(req *GetUsersRequest) (*GetUsersResponse, error)
	MockIsParticipant func(conversationID, participantID int64, internalToken string) (bool, error)
}

func (m *MockUserService) Name() string {
	if m.MockNameFunc != nil {
		return m.MockNameFunc()
	}

	return "Mock User Service"
}

func (m *MockUserService) Status() services.ServiceStatus {
	if m.MockStatusFunc != nil {
		return m.MockStatusFunc()
	}

	// Default status is READY
	return services.ServiceReady
}

func (m *MockUserService) Close() error {
	if m.MockCloseFunc != nil {
		return m.MockCloseFunc()
	}

	return nil
}

func (m *MockUserService) GetUsers(req *GetUsersRequest) (*GetUsersResponse, error) {
	if m.MockGetUsers != nil {
		return m.MockGetUsers(req)
	}

	return &GetUsersResponse{
		UserMap: map[int64]model.ChatUser{},
	}, nil
}

func (m *MockUserService) IsParticipant(conversationID, participantID int64, internalToken string) (bool, error) {
	if m.MockIsParticipant != nil {
		return m.MockIsParticipant(conversationID, participantID, internalToken)
	}

	return false, nil
}
