package context

import (
	"dungtl2003/chat-app-message-service/internal/httpclient"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/snowflake"
	"dungtl2003/chat-app-message-service/internal/validate"
)

type AppContext struct {
	Validator         *validate.Validator
	Logger            *logging.LoggerWrapper
	MessageServiceURL string
	Client            *httpclient.HttpClient

	IdGeneratorService *snowflake.IdGeneratorService
	DatabaseService    *database.DatabaseService
	Services           []services.Service
}
