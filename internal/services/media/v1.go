package media

import (
	"bytes"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"path/filepath"
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

// New creates a new MediaServiceV1 instance. Remember to call Close() when done
// to release resources.
func NewMediaServiceV1(mediaURL string, opts *MediaServiceV1Options) (*MediaServiceV1, error) {
	var loggerWrapper *logging.LoggerWrapper
	var client *http.Client

	if opts != nil && opts.Logger == nil {
		logger, err := logging.NewLogger(logging.INFO, logging.TEXT)
		if err != nil {
			return nil, fmt.Errorf("Failed to create logger: %v", err)
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
		status:   services.READY,
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
	if s.status == services.STOPPED {
		s.logger.Errorfln("[%s] Service is stopped", s.Name())
		return s.status
	}

	resp, err := s.client.Get(fmt.Sprintf("%s/healthcheck", s.mediaURL))
	if err != nil {
		s.logger.Errorfln("[%s] Failed to check service status: %v", s.Name(), err)
		s.status = services.ERROR
		return s.status
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		s.logger.Errorfln("[%s] Failed to decode service health response: %v", s.Name(), err)
		s.status = services.ERROR
		return s.status
	}

	if healthResp.Status == UP {
		s.logger.Infofln("[%s] Service is up and running", s.Name())
		s.status = services.READY
	} else {
		s.logger.Errorfln("[%s] Service is down, status: %s", s.Name(), healthResp.Status)
		s.status = services.ERROR
	}

	return s.status
}

// Name returns the name of the service.
func (s *MediaServiceV1) Name() string {
	return "Media Service"
}

// Close stops the service and releases any resources it holds.
func (s *MediaServiceV1) Close() error {
	if s.status == services.STOPPED {
		s.logger.Errorfln("[%s] Service is already stopped", s.Name())
		return nil
	}

	s.logger.Infofln("[%s] Closing service", s.Name())
	s.status = services.STOPPED
	s.logger.Infofln("[%s] Service stopped", s.Name())
	return nil
}

func (s *MediaServiceV1) PostAsset(fileHeader multipart.FileHeader, token string) (*AssetPostResponse, error) {
	if s.status == services.STOPPED {
		return nil, fmt.Errorf("service is stopped")
	}
	if s.status != services.READY {
		s.logger.Warnfln("[%s] Service is not ready", s.Name())
	}

	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		s.logger.Errorfln("[%s] Failed to open uploaded file: %v", s.Name(), err)
		return nil, err
	}
	defer file.Close()

	// Create a buffer and multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create form file field "asset"
	part, err := writer.CreateFormFile(FORM_FIELD_ASSET, filepath.Base(fileHeader.Filename))
	if err != nil {
		s.logger.Errorfln("[%s] Failed to create form file: %v", s.Name(), err)
		return nil, err
	}

	// Copy file content to part
	_, err = io.Copy(part, file)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to copy file content: %v", s.Name(), err)
		return nil, err
	}

	// Close the writer to finalize form
	err = writer.Close()
	if err != nil {
		s.logger.Errorfln("[%s] Failed to close multipart writer: %v", s.Name(), err)
		return nil, err
	}

	headers := http.Header{
		"Authorization": {fmt.Sprintf("Bearer %s", token)},
		"Content-Type":  {writer.FormDataContentType()},
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/assets", s.mediaURL), body)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to create POST request: %v", s.Name(), err)
		return nil, err
	}
	req.Header = headers

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to send POST request: %v", s.Name(), err)
		return nil, err
	}
	defer resp.Body.Close()

	mediaPostResponse := &AssetPostResponse{
		StatusCode: resp.StatusCode,
	}

	if err := json.NewDecoder(resp.Body).Decode(&mediaPostResponse.Body); err != nil {
		s.logger.Errorfln("[%s] Failed to decode response body: %v", s.Name(), err)
		return nil, err
	}

	return mediaPostResponse, nil
}

func (s *MediaServiceV1) GetAssetBatch(params AssetBatchPostParams) (*AssetBatchPostResponse, error) {
	if s.status == services.STOPPED {
		return nil, fmt.Errorf("service is stopped")
	}
	if s.status != services.READY {
		s.logger.Warnfln("[%s] Service is not ready", s.Name())
	}

	headers := http.Header{
		"Content-Type": {"application/json"},
	}

	bodyBytes, err := json.Marshal(params)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to marshal query parameters: %v", s.Name(), err)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/assets/batch", s.mediaURL), io.Reader(bytes.NewBuffer(bodyBytes)))
	if err != nil {
		s.logger.Errorfln("[%s] Failed to create POST request: %v", s.Name(), err)
		return nil, err
	}
	req.Header = headers

	s.logger.Debugfln("[%s] Sending POST request to %s with headers: %v and body: %s", s.Name(), req.URL, req.Header, string(bodyBytes))
	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to send POST request: %v", s.Name(), err)
		return nil, err
	}
	defer resp.Body.Close()

	s.logger.Debugfln("[%s] Received response with status code: %d", s.Name(), resp.StatusCode)
	var responseBody AssetBatchPostResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		s.logger.Errorfln("[%s] Failed to decode response body: %v", s.Name(), err)
		return nil, err
	}

	response := &AssetBatchPostResponse{
		StatusCode: resp.StatusCode,
		Body:       responseBody,
	}

	s.logger.Debugfln("[%s] Received response: %v", s.Name(), response)
	return response, nil
}

func (s *MediaServiceV1) GetAsset(params AssetGetParams) (*AssetGetResponse, error) {
	if s.status == services.STOPPED {
		return nil, fmt.Errorf("service is stopped")
	}
	if s.status != services.READY {
		s.logger.Warnfln("[%s] Service is not ready", s.Name())
	}

	headers := http.Header{
		"Content-Type": {"application/json"},
	}

	method := http.MethodGet
	req, err := http.NewRequest(method, fmt.Sprintf("%s/assets/%d", s.mediaURL, params.AssetId), nil)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to create %s request: %v", s.Name(), method, err)
		return nil, err
	}
	req.Header = headers

	s.logger.Debugfln("[%s] Sending %s request to %s with headers: %v", s.Name(), method, req.URL, req.Header)
	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to send %s request: %v", s.Name(), err, method)
		return nil, err
	}
	defer resp.Body.Close()

	s.logger.Debugfln("[%s] Received response with status code: %d", s.Name(), resp.StatusCode)
	var responseBody AssetGetResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		s.logger.Errorfln("[%s] Failed to decode response body: %v", s.Name(), err)
		return nil, err
	}

	response := &AssetGetResponse{
		StatusCode: resp.StatusCode,
		Body:       responseBody,
	}

	s.logger.Debugfln("[%s] Received response: %v", s.Name(), response)
	return response, nil
}
