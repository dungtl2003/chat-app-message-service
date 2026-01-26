package media

import (
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services"
)

type MockMediaService struct {
	MockNameFunc   func() string
	MockStatusFunc func() services.ServiceStatus
	MockCloseFunc  func() error
	MockGetAssets  func(req MediaGetBatchRequest) (*MediaGetBatchResponse, error)
}

func (m *MockMediaService) Name() string {
	if m.MockNameFunc != nil {
		return m.MockNameFunc()
	}

	return "Mock Media Service"
}

func (m *MockMediaService) Status() services.ServiceStatus {
	if m.MockStatusFunc != nil {
		return m.MockStatusFunc()
	}

	// Default status is READY
	return services.ServiceReady
}

func (m *MockMediaService) Close() error {
	if m.MockCloseFunc != nil {
		return m.MockCloseFunc()
	}

	return nil
}

func (m *MockMediaService) GetAssets(req MediaGetBatchRequest) (*MediaGetBatchResponse, error) {
	if m.MockGetAssets != nil {
		return m.MockGetAssets(req)
	}

	// Default implementation returns an empty response
	return &MediaGetBatchResponse{
		Assets: make(map[string]model.Asset),
	}, nil
}
