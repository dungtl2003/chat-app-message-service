package api

import (
	"dungtl2003/chat-app-message-service/internal/config"
	"dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services/conversation"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/idgen"
	"dungtl2003/chat-app-message-service/internal/services/kafka"
	"dungtl2003/chat-app-message-service/internal/services/media"
	"dungtl2003/chat-app-message-service/internal/services/user"
)

type HandlerDeps struct {
	Validator *helper.Validator
	Logger    *logging.LoggerWrapper
	Config    *config.Config

	AssetConfirmEventChannel chan kafka.KMessage[kafka.AssetResourceConfirmEvent]

	MediaService        media.MediaService
	IdGeneratorService  idgen.IdGeneratorService
	UserService         user.UserService
	ConversationService conversation.ConversationService
	DatabaseService     *database.DatabaseService
}
