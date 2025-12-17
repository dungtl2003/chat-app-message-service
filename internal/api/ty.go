package api

import (
	"dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/idgen"
	"dungtl2003/chat-app-message-service/internal/services/kafka"
	"dungtl2003/chat-app-message-service/internal/services/media"
)

type HandlerDeps struct {
	Validator *helper.Validator
	Logger    *logging.LoggerWrapper

	MediaService       media.MediaService
	IdGeneratorService idgen.IdGeneratorService
	DatabaseService    *database.DatabaseService
	KafkaProducer      *kafka.KafkaProducer
}
