package media

import (
	"bytes"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services"
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type MediaServiceV1 struct {
	status   services.ServiceStatus
	client   *http.Client
	logger   *logging.LoggerWrapper
	mediaURL string
}

type MediaServiceV1Options struct {
	Logger *logging.LoggerWrapper
	Client *http.Client
}

// New creates a new RealMediaService instance. Remember to call Close() when done
// to release resources.
func NewMediaServiceV1(mediaURL string, opts *MediaServiceV1Options) (*MediaServiceV1, error) {
	var loggerWrapper *logging.LoggerWrapper
	var client *http.Client

	if opts != nil && opts.Logger == nil {
		logger, err := logging.NewLogger(logging.INFO, logging.TEXT)
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %v", err)
		}
		loggerWrapper = logging.NewLoggerWrapper(logger)
	} else {
		loggerWrapper = opts.Logger
	}

	if opts != nil && opts.Client == nil {
		client = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:          100,              // Max idle connections across all hosts
				MaxIdleConnsPerHost:   10,               // Max idle connections per host
				IdleConnTimeout:       90 * time.Second, // Keep idle connections for 90s
				TLSHandshakeTimeout:   10 * time.Second, // Timeout for TLS handshake
				ExpectContinueTimeout: 1 * time.Second,  // Wait time for 100-Continue responses
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,  // Connection timeout
					KeepAlive: 30 * time.Second, // TCP keep-alive time
				}).DialContext,
			},
			Timeout: 15 * time.Second, // Overall request timeout
		}
	} else {
		client = opts.Client
	}

	service := &MediaServiceV1{
		status:   services.ServiceReady,
		client:   client,
		logger:   loggerWrapper,
		mediaURL: mediaURL,
	}

	service.logger.Infofln("[%s] Service created with URL: %s", service.Name(), mediaURL)
	return service, nil
}

// Status checks the health of the media service by making a request to the
// healthcheck endpoint. It returns the service status based on the response.
func (s *MediaServiceV1) Status() services.ServiceStatus {
	if s.status == services.ServiceStopped {
		s.logger.Errorfln("[%s] Service is stopped", s.Name())
		return s.status
	}
	if s.status != services.ServiceReady {
		s.logger.Warnfln("[%s] Service is not ready", s.Name())
	}

	resp, err := s.client.Get(fmt.Sprintf("%s/healthcheck", s.mediaURL))
	if err != nil {
		s.logger.Errorfln("[%s] Failed to check service status: %v", s.Name(), err)
		s.status = services.ServiceError
		return s.status
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		s.logger.Errorfln("[%s] Failed to decode service health response: %v", s.Name(), err)
		s.status = services.ServiceError
		return s.status
	}

	if healthResp.Status == UP {
		s.logger.Infofln("[%s] Service is up and running", s.Name())
		s.status = services.ServiceReady
	} else {
		s.logger.Errorfln("[%s] Service is down, status: %s", s.Name(), healthResp.Status)
		s.status = services.ServiceError
	}

	return s.status
}

// Name returns the name of the media service.
func (s *MediaServiceV1) Name() string {
	return "Media Service V1"
}

// Close stops the media service and releases any resources it holds.
func (s *MediaServiceV1) Close() error {
	if s.status == services.ServiceStopped {
		s.logger.Errorfln("[%s] Service is already stopped", s.Name())
		return nil
	}

	s.logger.Infofln("[%s] Closing service", s.Name())
	s.status = services.ServiceStopped
	s.logger.Infofln("[%s] Service stopped", s.Name())
	return nil
}

func (s *MediaServiceV1) GetAssets(req MediaGetBatchRequest) (*MediaGetBatchResponse, error) {
	type GetAssetsMapRequestBody struct {
		AssetIds []types.JsonInt64 `json:"asset_ids" validate:"required"`
	}

	type GetAssetsMapResponseBody struct {
		Assets map[string]model.Asset `json:"assets"`
	}

	if s.status == services.ServiceStopped {
		return nil, fmt.Errorf("service is stopped")
	}
	if s.status != services.ServiceReady {
		s.logger.Warnfln("[%s] Service is not ready", s.Name())
	}

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", req.InternalToken)},
	}
	method := http.MethodPost

	var requestBody = &GetAssetsMapRequestBody{
		AssetIds: req.AssetIds,
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	httpReq, err := http.NewRequest(method, fmt.Sprintf("%s/assets/map", s.mediaURL), bytes.NewReader(requestBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	httpReq.Header = headers

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %v", err)
	}
	defer resp.Body.Close()

	var respBody types.Response[GetAssetsMapResponseBody]
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %v", err)
	}

	if respBody.Error != nil {
		return nil, BadResponseError{ErrBlock: *respBody.Error}
	}

	return &MediaGetBatchResponse{
		Assets: respBody.Data.Item.Assets,
	}, nil
}
