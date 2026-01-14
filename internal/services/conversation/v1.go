package conversation

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

type ConversationServiceV1 struct {
	logger                  *logging.LoggerWrapper
	status                  services.ServiceStatus
	client                  *http.Client
	conversationAPIEndpoint string
}

type ConversationServiceV1Options struct {
	Logger *logging.LoggerWrapper
	Client *http.Client
}

func NewConversationServiceV1(conversationAPIEndpoint string, opts *ConversationServiceV1Options) (*ConversationServiceV1, error) {
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

	service := &ConversationServiceV1{
		status:                  services.ServiceReady,
		client:                  client,
		logger:                  loggerWrapper,
		conversationAPIEndpoint: conversationAPIEndpoint,
	}

	service.logger.Infofln("[%s] Service created with URL: %s", service.Name(), conversationAPIEndpoint)
	return service, nil
}

func (s *ConversationServiceV1) BatchGetParticipants(req *BatchGetParticipantsRequest) (*BatchGetParticipantsResponse, error) {
	type RequestBody struct {
		ConversationID types.JsonInt64   `json:"conversation_id"`
		UserIDs        []types.JsonInt64 `json:"user_ids"`
	}
	type ResponseBody struct {
		ParticipantMap map[string]model.Participant `json:"participant_map"`
	}

	if s.status == services.ServiceStopped {
		return nil, services.ServiceNotRunningError{ServiceName: s.Name()}
	}
	if s.status != services.ServiceReady {
		s.logger.Warnfln("[%s] Service is not ready", s.Name())
	}

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", req.InternalToken)},
	}
	url := fmt.Sprintf("%s/conversations/participants/batch-get", s.conversationAPIEndpoint)
	method := http.MethodPost
	body := RequestBody{
		ConversationID: types.NewJsonInt64(req.ConversationID),
		UserIDs:        make([]types.JsonInt64, len(req.UserIDs)),
	}
	for i, id := range req.UserIDs {
		body.UserIDs[i] = types.NewJsonInt64(id)
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to marshal request body: %v", s.Name(), err)
		return nil, err
	}

	reqHTTP, err := http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		s.logger.Errorfln("[%s] Failed to create %s request: %v", s.Name(), method, err)
		return nil, err
	}
	reqHTTP.Header = headers

	s.logger.Debugfln("[%s] Sending %s request to %s with headers: %v", s.Name(), method, reqHTTP.URL, reqHTTP.Header)
	resp, err := s.client.Do(reqHTTP)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to send %s request: %v", s.Name(), method, err)
		return nil, err
	}
	defer resp.Body.Close()

	var respBody types.Response[ResponseBody]
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		s.logger.Errorfln("[%s] Failed to decode response body: %v", s.Name(), err)
		return nil, err
	}
	if respBody.Error != nil {
		s.logger.Errorfln("[%s] Received error response: %v", s.Name(), respBody.Error)
		return nil, BadResponseError{ErrBlock: *respBody.Error}
	}

	participantMap := make(map[int64]model.Participant)
	for idStr, participant := range respBody.Data.Item.ParticipantMap {
		var id int64
		if _, err := fmt.Sscan(idStr, &id); err != nil {
			s.logger.Errorfln("[%s] Failed to parse user ID %s: %v", s.Name(), idStr, err)
			return nil, err
		}
		participantMap[id] = participant
	}

	s.logger.Infofln("[%s] Successfully retrieved %d participants", s.Name(), len(participantMap))
	return &BatchGetParticipantsResponse{
		ParticipantMap: participantMap,
	}, nil
}

func (s *ConversationServiceV1) IsParticipant(conversationID, participantID int64, internalToken string) (bool, error) {
	type ResponseBody struct {
		IsMember bool `json:"is_member"`
	}

	if s.status == services.ServiceStopped {
		return false, services.ServiceNotRunningError{ServiceName: s.Name()}
	}
	if s.status != services.ServiceReady {
		s.logger.Warnfln("[%s] Service is not ready", s.Name())
	}

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", internalToken)},
	}
	url := fmt.Sprintf("%s/conversations/%d/participants/%d/membership", s.conversationAPIEndpoint, conversationID, participantID)
	method := http.MethodGet

	reqHTTP, err := http.NewRequest(method, url, nil)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to create %s request: %v", s.Name(), method, err)
		return false, err
	}
	reqHTTP.Header = headers

	s.logger.Debugfln("[%s] Sending %s request to %s with headers: %v", s.Name(), method, reqHTTP.URL, reqHTTP.Header)
	resp, err := s.client.Do(reqHTTP)
	if err != nil {
		s.logger.Errorfln("[%s] Failed to send %s request: %v", s.Name(), method, err)
		return false, err
	}
	defer resp.Body.Close()

	var respBody types.Response[ResponseBody]
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		s.logger.Errorfln("[%s] Failed to decode response body: %v", s.Name(), err)
		return false, err
	}
	if respBody.Error != nil {
		s.logger.Errorfln("[%s] Received error response: %v", s.Name(), respBody.Error)
		return false, BadResponseError{ErrBlock: *respBody.Error}
	}

	s.logger.Infofln("[%s] Successfully retrieved participant status for conversation %d and participant %d: %v", s.Name(), conversationID, participantID, respBody.Data.Item.IsMember)
	return respBody.Data.Item.IsMember, nil
}

// Status checks the health of the service by making a request to the
// healthcheck endpoint. It returns the service status based on the response.
func (s *ConversationServiceV1) Status() services.ServiceStatus {
	if s.status == services.ServiceStopped {
		s.logger.Errorfln("[%s] Service is stopped", s.Name())
		return s.status
	}
	if s.status != services.ServiceReady {
		s.logger.Warnfln("[%s] Service is not ready", s.Name())
	}

	resp, err := s.client.Get(fmt.Sprintf("%s/healthcheck", s.conversationAPIEndpoint))
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

// Name returns the name of the service.
func (s *ConversationServiceV1) Name() string {
	return "Conversation Service V1"
}

// Close closes the connection to the service.
func (s *ConversationServiceV1) Close() error {
	if s.status == services.ServiceStopped {
		s.logger.Errorfln("[%s] Service is already stopped", s.Name())
		return nil
	}

	s.logger.Infofln("[%s] Closing service", s.Name())
	s.status = services.ServiceStopped
	s.logger.Infofln("[%s] Service stopped", s.Name())
	return nil
}
