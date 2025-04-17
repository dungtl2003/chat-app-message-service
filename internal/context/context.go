package context

import (
	"dungtl2003/chat-app-message-service/internal/database"
	"dungtl2003/chat-app-message-service/internal/httpclient"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services/snowflake"
	"dungtl2003/chat-app-message-service/internal/validate"
)

type AppContext struct {
	IdGeneratorService *snowflake.IdGeneratorService
	Validator          *validate.Validator
	Logger             *logging.LoggerWrapper
	Database           *database.Database
	MessageServiceURL  string
	Client             *httpclient.HttpClient
}
