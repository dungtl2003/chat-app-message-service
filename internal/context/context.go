package context

import (
	"dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/idgen"
)

type AppContext struct {
	Validator         *helper.Validator
	Logger            *logging.LoggerWrapper
	MessageServiceURL string

	IdGeneratorService idgen.IdGeneratorService
	DatabaseService    *database.DatabaseService
	Services           []services.Service
}
