package api

import (
	ctx "context"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AttachmentPostRequestBody struct {
	AssetId  types.JsonInt64      `json:"asset_id" validate:"required"`
	Position int                  `json:"position" validate:"required"`
	Type     model.AttachmentType `json:"type" validate:"required,oneof=IMAGE VIDEO AUDIO FILE GIF STICKER"`
}

type MessagePostRequestBody struct {
	SenderId         types.JsonInt64     `json:"sender_id" validate:"required"`
	ReceiverId       types.JsonInt64     `json:"receiver_id" validate:"required"`
	Content          string              `json:"content" validate:"required"`
	Type             model.MessageType   `json:"type" validate:"required"`
	ReplyToMessageId types.JsonNullInt64 `json:"reply_to_message_id"`

	Attachments []AttachmentPostRequestBody `json:"attachments"`
}

func (a AttachmentPostRequestBody) String() string {
	b, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(b)
}

func (m MessagePostRequestBody) String() string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

func GetMessagesByConvID(handlerDeps *HandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := types.Response[model.Message]{}

		conversationIdStr := c.Param("conversation-id")
		afterStr := c.Query("after")
		limitStr := c.Query("limit")
		delayStr := c.Query("delay")

		handlerDeps.Logger.Debugfln("query(conversation_id=%s, after=%s, limit=%s)", conversationIdStr, afterStr, limitStr)

		conversationId, err := strconv.ParseInt(conversationIdStr, 10, 64)
		if err != nil {
			handlerDeps.Logger.Errorfln("error parsing conversation ID: %v", err)
			resp.Error = &types.ErrorBlock{
				Code:    http.StatusBadRequest,
				Message: "Invalid conversation ID",
				Errors: []types.ErrorItem{{
					Message: "Invalid conversation ID",
				}},
			}

			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}

		optionalAfter := types.NewOptional[int64]()
		if afterStr != "" {
			after, err := strconv.ParseInt(afterStr, 10, 64)
			if err != nil {
				handlerDeps.Logger.Errorfln("error parsing after query: %v", err)
				resp.Error = &types.ErrorBlock{
					Code:    http.StatusBadRequest,
					Message: "Invalid after query",
					Errors: []types.ErrorItem{{
						Message: "Invalid after query",
					}},
				}

				c.JSON(resp.Error.Code, resp)
				c.Abort()
				return
			}

			if after < 0 {
				handlerDeps.Logger.Error("after query cannot be less than 0")
				resp.Error = &types.ErrorBlock{
					Code:    http.StatusBadRequest,
					Message: "After query cannot be less than 0",
					Errors: []types.ErrorItem{{
						Message: "After query cannot be less than 0",
					}},
				}

				c.JSON(resp.Error.Code, resp)
				c.Abort()
				return
			}

			optionalAfter.SetValue(after)
		}

		optionalLimit := types.NewOptional[int64]()
		if limitStr != "" {
			limit, err := strconv.ParseInt(limitStr, 10, 64)
			if err != nil {
				handlerDeps.Logger.Errorfln("error parsing limit query: %v", err)
				resp.Error = &types.ErrorBlock{
					Code:    http.StatusBadRequest,
					Message: "Invalid limit query",
					Errors: []types.ErrorItem{{
						Message: "Invalid limit query",
					}},
				}

				c.JSON(resp.Error.Code, resp)
				c.Abort()
				return
			}

			if limit < 0 {
				handlerDeps.Logger.Error("limit query cannot be less than 0")
				resp.Error = &types.ErrorBlock{
					Code:    http.StatusBadRequest,
					Message: "Limit query cannot be less than 0",
					Errors: []types.ErrorItem{{
						Message: "Limit query cannot be less than 0",
					}},
				}

				c.JSON(resp.Error.Code, resp)
				c.Abort()
				return
			}

			optionalLimit.SetValue(limit)
		}

		optionalDelay := types.NewOptional[int64]()
		if delayStr != "" {
			delay, err := strconv.ParseInt(delayStr, 10, 64)
			if err != nil {
				handlerDeps.Logger.Errorfln("error parsing delay query: %v", err)
				resp.Error = &types.ErrorBlock{
					Code:    http.StatusBadRequest,
					Message: "Invalid delay query",
					Errors: []types.ErrorItem{{
						Message: "Invalid delay query",
					}},
				}

				c.JSON(resp.Error.Code, resp)
				c.Abort()
				return
			}
			if delay < 0 {
				handlerDeps.Logger.Errorfln("delay query cannot be less than 0")
				resp.Error = &types.ErrorBlock{
					Code:    http.StatusBadRequest,
					Message: "Delay query cannot be less than 0",
					Errors: []types.ErrorItem{{
						Message: "Delay query cannot be less than 0",
					}},
				}

				c.JSON(resp.Error.Code, resp)
				c.Abort()
				return
			}
			optionalDelay.SetValue(delay)
		}
		if optionalDelay.Valid {
			handlerDeps.Logger.Debugfln("Delaying response for %d ms", optionalDelay.Value)
			// Simulate delay
			time.Sleep(time.Duration(optionalDelay.Value) * time.Millisecond)
		}

		cursor, messages, hasMore, err := handlerDeps.DatabaseService.GetMessages(conversationId, *optionalAfter, *optionalLimit)
		if err != nil {
			handlerDeps.Logger.Errorfln("Database.GetMessages(): %v", err)
			resp.Error = &types.ErrorBlock{
				Message: err.Error(),
				Errors: []types.ErrorItem{{
					Message: err.Error(),
				}},
			}

			switch err {
			case database.ErrDatabaseError:
				resp.Error.Code = http.StatusInternalServerError
			default:
				resp.Error.Code = http.StatusInternalServerError
			}

			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}

		resp.Data = &types.DataOrPage[model.Message]{}
		resp.Data.Page = &types.Page[model.Message]{
			Items: messages,
		}
		resp.Data.Page.CurrentItemCount = types.NewJsonNullInt64(int64(len(messages)))
		resp.Data.Page.EndCursor = fmt.Sprintf("%d", cursor)
		resp.Data.Page.HasMore = hasMore

		if optionalLimit.Valid {
			resp.Data.Page.ItemsPerPage = types.NewJsonNullInt64(optionalLimit.Value)
		}

		handlerDeps.Logger.Debugfln("response: %s", resp)
		c.JSON(http.StatusOK, resp)
		c.Abort()
	}
}

// GET /messages/:message-id
func GetMessageByID(handlerDeps *HandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := types.Response[model.Message]{}

		messageIdStr := c.Param("message-id")
		handlerDeps.Logger.Debugfln("query(message_id=%s)", messageIdStr)

		messageId, err := strconv.ParseInt(messageIdStr, 10, 64)
		if err != nil {
			handlerDeps.Logger.Errorfln("error parsing message ID: %v", err)
			resp.Error = &types.ErrorBlock{
				Code:    http.StatusBadRequest,
				Message: "Invalid message ID",
				Errors: []types.ErrorItem{{
					Message: "Invalid message ID",
				}},
			}

			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}

		message, err := handlerDeps.DatabaseService.GetMessageById(messageId)
		if err != nil {
			handlerDeps.Logger.Errorfln("Database.GetMessageById(): %v", err)
			resp.Error = &types.ErrorBlock{
				Message: err.Error(),
				Errors: []types.ErrorItem{{
					Message: err.Error(),
				}},
			}

			switch err {
			case database.ErrDatabaseError:
				resp.Error.Code = http.StatusInternalServerError
			default:
				resp.Error.Code = http.StatusInternalServerError
			}

			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}

		resp.Data = &types.DataOrPage[model.Message]{}
		resp.Data.Item = message

		handlerDeps.Logger.Debugfln("response: %s", resp)
		c.JSON(http.StatusOK, resp)
		c.Abort()
	}
}

func CreateMessage(handlerDeps *HandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := types.Response[model.Message]{}
		var reqBody MessagePostRequestBody
		if err := c.ShouldBindJSON(&reqBody); err != nil {
			handlerDeps.Logger.Errorfln("c.ShouldBindJSON(): %v", err)
			resp.Error = &types.ErrorBlock{
				Code:    http.StatusBadRequest,
				Message: "Invalid request body",
				Errors: []types.ErrorItem{{
					Message: "Invalid request body",
				}},
			}

			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}
		if err := handlerDeps.Validator.Validate(reqBody); err != nil {
			handlerDeps.Logger.Errorfln("Validator.Validate(): %v", err)
			resp.Error = &types.ErrorBlock{
				Code:    http.StatusBadRequest,
				Message: "Invalid request body",
				Errors: []types.ErrorItem{{
					Message: "Invalid request body",
				}},
			}

			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}
		handlerDeps.Logger.Debugfln("request body: %s", reqBody)

		timeoutContext, cancel := ctx.WithTimeout(c, 5*time.Second)
		defer cancel()
		messageId, err := handlerDeps.IdGeneratorService.GenerateId(timeoutContext)
		if err != nil {
			handlerDeps.Logger.Errorfln("IdGeneratorService.GenerateId(): %v", err)
			resp.Error = &types.ErrorBlock{
				Code:    http.StatusInternalServerError,
				Message: "Internal server error",
				Errors: []types.ErrorItem{{
					Message: "Internal server error",
				}},
			}

			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}

		attachments := make([]model.Attachment, len(reqBody.Attachments))
		for i, attachment := range reqBody.Attachments {
			timeoutContext, cancel := ctx.WithTimeout(c, 5*time.Second)
			defer cancel()
			attachmentId, err := handlerDeps.IdGeneratorService.GenerateId(timeoutContext)
			if err != nil {
				handlerDeps.Logger.Errorfln("IdGeneratorService.GenerateId(): %v", err)
				resp.Error = &types.ErrorBlock{
					Code:    http.StatusInternalServerError,
					Message: "Internal server error",
					Errors: []types.ErrorItem{{
						Message: "Internal server error",
					}},
				}

				c.JSON(resp.Error.Code, resp)
				c.Abort()
				return
			}
			attachments[i] = model.Attachment{
				Id:        types.NewJsonInt64(attachmentId),
				AssetId:   attachment.AssetId,
				MessageId: types.NewJsonInt64(messageId),
				Position:  attachment.Position,
				Type:      attachment.Type,
			}
		}

		message := model.Message{
			Id:               types.NewJsonInt64(messageId),
			Content:          reqBody.Content,
			Type:             reqBody.Type,
			CreatedAt:        types.NewJsonTime(time.Now().UTC()),
			SenderId:         reqBody.SenderId,
			ReceiverId:       reqBody.ReceiverId,
			Attachments:      attachments,
			ReplyToMessageId: reqBody.ReplyToMessageId,
		}
		handlerDeps.Logger.Debugfln("message: %s", message)

		msg, err := handlerDeps.DatabaseService.CreateMessage(message)
		if err != nil {
			handlerDeps.Logger.Errorfln("Database.CreateMessage(): %v", err)
			resp.Error = &types.ErrorBlock{
				Message: err.Error(),
				Errors: []types.ErrorItem{{
					Message: err.Error(),
				}},
			}

			switch err {
			case database.ErrDatabaseError:
				resp.Error.Code = http.StatusInternalServerError
			default:
				resp.Error.Code = http.StatusInternalServerError
			}

			c.JSON(resp.Error.Code, resp)
			c.Abort()
			return
		}

		// go func() {
		// 	err := kafka.WriteMessages(handlerDeps.KafkaWriterService, []kafka.KMessage[kafka.MessageResourceCreatedEvent]{
		// 		{
		// 			Topic: kafka.MESSAGE_RESOURCE_CREATED_TOPIC,
		// 			Key:   kafka.CreateEventKey(msg.ReceiverId.Int64()),
		// 			Value: kafka.MessageResourceCreatedEvent{
		// 				Message: *msg,
		// 			},
		// 		},
		// 	})
		//
		// 	if err != nil {
		// 		handlerDeps.Logger.Errorfln("failed to write message resource created event to kafka: %v", err)
		// 	} else {
		// 		handlerDeps.Logger.Debugfln("message resource created event written to kafka successfully")
		// 	}
		// }()

		resp.Data = &types.DataOrPage[model.Message]{}
		resp.Data.Item = msg

		c.JSON(http.StatusCreated, resp)
		c.Abort()
	}
}
