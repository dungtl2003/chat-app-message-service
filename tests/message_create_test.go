package tests

import (
	"bytes"
	"context"
	"dungtl2003/chat-app-message-service/internal/api"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/server"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/kafka"
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	gokafka "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

const (
	USERS__MSG__CREATE_FILENAME = "chat_users__message__create_test.json"
	CONVS__MSG__CREATE_FILENAME = "conversations__message__create_test.json"
)

func TestMessageCreateFlowShouldWork(t *testing.T) {
	helper := NewTestHelper()
	SetUp(helper, &SetUpOptions{
		DataFile: &database.DataFile{
			UserFile:         USERS__MSG__CREATE_FILENAME,
			ConversationFile: CONVS__MSG__CREATE_FILENAME,
		},
		ServerOptions: &server.MessageServerOptions{},
	})
	defer TearDown(helper)

	// We create a new reader (consumer) to consume the message from the kafka broker
	// We want to verify that the message is actually sent to the kafka broker
	reader := gokafka.NewReader(gokafka.ReaderConfig{
		Brokers:     []string{helper.BrokerAddr},
		Topic:       string(kafka.MESSAGE_RESOURCE_CREATED_TOPIC),
		GroupID:     KAFKA_GROUP_ID,
		StartOffset: gokafka.FirstOffset,
	})
	defer reader.Close()

	senderId := int64(4) // user 4 in group 2
	receiverId := int64(2)
	internalToken := GetInternalAccessToken(senderId)
	header := http.Header{
		"Authorization": []string{"Bearer " + internalToken},
		"Cache-Control": []string{"no-cache"},
	}
	url := fmt.Sprintf("%s/messages", helper.MessageServiceURL)
	reqBody := api.MessagePostRequestBody{
		SenderId:       types.NewJsonInt64(senderId),
		ReceiverId:     types.NewJsonInt64(receiverId),
		Content:        "Hello from user 4 to user 2",
		Type:           model.MSG_TEXT,
		IdempotencyKey: "unique-key-12345",
	}
	reqBodyJson, err := json.Marshal(reqBody)
	require.NoError(t, err)

	resp, err := Post(helper.Client, url, header, bytes.NewBuffer(reqBodyJson))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var respBody types.Response[model.Message]
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)

	require.Nil(t, respBody.Error)
	require.NotNil(t, respBody.Data)
	require.NotNil(t, respBody.Data.Item)

	respMsg := respBody.Data.Item
	require.Equal(t, reqBody.Content, respMsg.Content)
	require.Equal(t, reqBody.IdempotencyKey, respMsg.IdempotencyKey)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait for the message to be received
	msg, err := reader.FetchMessage(ctx)
	require.NoError(t, err, "failed to fetch message from kafka")

	var event kafka.MessageResourceCreatedEvent
	err = json.Unmarshal(msg.Value, &event)
	require.NoError(t, err)
	require.EqualValues(t, respMsg.Id.Int64(), event.Message.Id.Int64())
	require.EqualValues(t, respMsg.Content, event.Message.Content)
	require.EqualValues(t, respMsg.IdempotencyKey, event.IdempotencyKey)

	err = reader.CommitMessages(context.Background(), msg)
	require.NoError(t, err)

	// Check the outbox event status after commit
	// It should be SENT
	require.Eventually(t, func() bool {
		helper.Logger.Infofln("Checking outbox event %d status after commit", event.OutboxId.Int64())
		outboxEvent, err := helper.AdminDatabaseService.GetOutboxEventById(context.Background(), event.OutboxId.Int64())
		if err != nil {
			return false
		}
		return outboxEvent.Status == model.OUTBOX_SENT
	}, 10*time.Second, 100*time.Millisecond, "outbox event should be processed after commit")
}
