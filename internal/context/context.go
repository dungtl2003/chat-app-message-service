package context

import (
	"dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/idgen"
	"dungtl2003/chat-app-message-service/internal/services/kafka"
)

type AppContext struct {
	Validator  *helper.Validator
	Logger     *logging.LoggerWrapper
	DlqChannel chan kafka.KMessage[kafka.DLQEvent]

	IdGeneratorService idgen.IdGeneratorService
	DatabaseService    *database.DatabaseService
	KafkaWriterService *kafka.KafkaWriterService
	Services           []services.Service
}
