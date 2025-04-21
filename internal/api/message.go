package api

import (
	"dungtl2003/chat-app-message-service/internal/context"
	"dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/types"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type PostAttachmentRequestBody struct {
	ThumbURL string `json:"thumb_url"`
	FileURL  string `json:"file_url"`
}

type PostMessageRequestBody struct {
	SenderId   types.JsonInt64   `json:"sender_id,required"`
	ReceiverId types.JsonInt64   `json:"receiver_id,required"`
	Content    string            `json:"content,required"`
	Type       model.MessageType `json:"type,required"`

	Attachments []PostAttachmentRequestBody `json:"attachments"`
}

func GetMessages(appCtx *context.AppContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		conversationIdStr := c.Query("conversation_id")
		afterStr := c.Query("after")
		limitStr := c.Query("limit")
		orderBy := c.Query("order_by")

		appCtx.Logger.Debugfln("query(conversation_id=%s, after=%s, limit=%s, order_by=%s)", conversationIdStr, afterStr, limitStr, orderBy)

		conversationId, err := strconv.ParseInt(conversationIdStr, 10, 64)
		if err != nil {
			appCtx.Logger.Debugfln("error parsing conversation ID: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
			c.Abort()
			return
		}

		optionalAfter := types.NewOptional[int64]()
		if afterStr != "" {
			after, err := strconv.ParseInt(afterStr, 10, 64)
			if err != nil {
				appCtx.Logger.Debugfln("error parsing after query: %v", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid after query"})
				c.Abort()
				return
			}

			if after < 0 {
				appCtx.Logger.Debug("after query cannot be less than 0")
				c.JSON(http.StatusBadRequest, gin.H{"error": "After query cannot be less than 0"})
				c.Abort()
				return
			}

			optionalAfter.SetValue(after)
		}

		optionalLimit := types.NewOptional[int64]()
		if limitStr != "" {
			limit, err := strconv.ParseInt(limitStr, 10, 64)
			if err != nil {
				appCtx.Logger.Debugfln("error parsing limit query: %v", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit query"})
				c.Abort()
				return
			}

			if limit < 0 {
				appCtx.Logger.Debug("limit query cannot be less than 0")
				c.JSON(http.StatusBadRequest, gin.H{"error": "Limit query cannot be less than 0"})
				c.Abort()
				return
			}

			optionalLimit.SetValue(limit)
		}

		optionalOrderBy := types.NewOptional[string]()
		if orderBy != "" {
			parts := strings.Split(orderBy, ":")
			if len(parts) != 2 {
				appCtx.Logger.Debugfln("order_by must have [key]:[value] structure, found %s", orderBy)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Order_by must have [key]:[value] structure"})
				c.Abort()
				return
			}

			key := parts[0]
			order := parts[1]
			allowedKeys := helper.GetStructFieldsInSnakeCase(model.Message{})
			allowedOrders := []string{"asc", "desc"}

			if !slices.Contains(allowedKeys, key) {
				allowedKeysStr := strings.Join(allowedKeys, ", ")
				appCtx.Logger.Debugfln("order_by=`[key]:[value]`: [key] must be one of the following: %s", allowedKeysStr)
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("order_by=`[key]:[value]`: [key] must be one of the following: %s", allowedKeysStr)})
				c.Abort()
				return
			}

			if !slices.Contains(allowedOrders, order) {
				allowedOrdersStr := strings.Join(allowedOrders, ", ")
				appCtx.Logger.Debugfln("[key] must be one of the following: %s", allowedOrdersStr)
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("order_by=`[key]:[value]`: [key] must be one of the following: %s", allowedOrdersStr)})
				c.Abort()
				return
			}

			optionalOrderBy.SetValue(fmt.Sprintf("%s:%s", key, strings.ToUpper(order)))
		}

		messages, hasMore, status, err := appCtx.Database.GetMessages(conversationId, *optionalAfter, *optionalLimit, *optionalOrderBy)
		if err != nil {
			appCtx.Logger.Debugfln("Database.GetMessages(): %v", err)
			c.JSON(status, gin.H{"error": fmt.Sprintf("%v", err)})
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": messages, "has_more": hasMore})
		c.Abort()
	}
}

func CreateMessage(appCtx *context.AppContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var postMessageRequestBody PostMessageRequestBody
		if err := c.ShouldBindJSON(&postMessageRequestBody); err != nil {
			appCtx.Logger.Debugfln("c.ShouldBindJSON(): %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			c.Abort()
			return
		}
		if err := appCtx.Validator.Validate(postMessageRequestBody); err != nil {
			appCtx.Logger.Debugfln("Validator.Validate(): %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			c.Abort()
			return
		}
		appCtx.Logger.Debugfln("request body: %#v", postMessageRequestBody)

		messageId, err := appCtx.IdGeneratorService.GenerateId()
		if err != nil {
			appCtx.Logger.Errorfln("IdGeneratorService.GenerateId(): %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		attachments := make([]model.Attachment, len(postMessageRequestBody.Attachments))
		for i, attachment := range postMessageRequestBody.Attachments {
			attachmentId, err := appCtx.IdGeneratorService.GenerateId()
			if err != nil {
				appCtx.Logger.Errorfln("IdGeneratorService.GenerateId(): %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				c.Abort()
				return
			}
			attachments[i] = model.Attachment{
				Id:        types.NewJsonInt64(attachmentId),
				ThumbURL:  attachment.ThumbURL,
				FileURL:   attachment.FileURL,
				DeletedAt: types.DefaultJsonNullTime(),
			}
		}

		message := model.Message{
			Id:          types.NewJsonInt64(messageId),
			Content:     postMessageRequestBody.Content,
			Type:        postMessageRequestBody.Type,
			CreatedAt:   types.NewJsonTime(time.Now().UTC()),
			UpdatedAt:   types.DefaultJsonNullTime(),
			DeletedAt:   types.DefaultJsonNullTime(),
			SenderId:    postMessageRequestBody.SenderId,
			ReceiverId:  postMessageRequestBody.ReceiverId,
			Attachments: attachments,
		}
		appCtx.Logger.Debugfln("message: %#v", message)

		msg, status, err := appCtx.Database.CreateMessage(message)
		if err != nil {
			appCtx.Logger.Debugfln("Database.CreateMessage(): %v", err)
			c.JSON(status, gin.H{"error": fmt.Sprintf("%v", err)})
			c.Abort()
			return
		}

		c.JSON(status, msg)
		c.Abort()
	}
}
