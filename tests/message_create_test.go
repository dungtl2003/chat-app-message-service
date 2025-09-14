package tests

import (
	"bytes"
	"context"
	"dungtl2003/chat-app-message-service/internal/api"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/kafka"
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	kk "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

const (
	USERS__MESSAGE__CREATE_FILENAME  = "chat_users__message__create_test.json"
	CONVS__MESSAGE__CREATE_FILENAME  = "conversations__message__create_test.json"
	ASSETS__MESSAGE__CREATE_FILENAME = "assets__message__create_test.json"
)

func TestCreateMessageShouldWork(t *testing.T) {
	helper := NewTestHelper()
	SetUp(helper, &SetUpOptions{
		DataFile: &database.DataFile{
			UserFile:         USERS__MESSAGE__CREATE_FILENAME,
			ConversationFile: CONVS__MESSAGE__CREATE_FILENAME,
			AssetFile:        ASSETS__MESSAGE__CREATE_FILENAME,
		},
	})
	defer TearDown(helper)

	token := GetInternalAccessToken(2)
	payload := api.MessagePostRequestBody{
		SenderId:   types.NewJsonInt64(4),
		ReceiverId: types.NewJsonInt64(2),
		Content:    "Hey! Check out these files.",
		Type:       model.MSG_IMAGE,

		Attachments: []api.AttachmentPostRequestBody{
			{
				AssetId:  types.NewJsonInt64(1),
				Position: 0,
				Type:     model.ATT_IMAGE,
			},
			{
				AssetId:  types.NewJsonInt64(2),
				Position: 1,
				Type:     model.ATT_IMAGE,
			},
			{
				AssetId:  types.NewJsonInt64(3),
				Position: 2,
				Type:     model.ATT_IMAGE,
			},
		},
	}
	payloadJson, err := json.Marshal(payload)
	require.NoError(t, err)

	url := fmt.Sprintf("%s/messages", helper.MessageServiceURL)
	header := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", token)},
	}
	resp, err := Post(helper.Client, url, header, bytes.NewBuffer(payloadJson))
	require.NoError(t, err)
	require.EqualValues(t, http.StatusCreated, resp.StatusCode)

	var respBody types.Response[model.Message]
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)

	require.Empty(t, respBody.Error)
	require.NotEmpty(t, respBody.Data)
	require.Nil(t, respBody.Data.Page)
	require.NotNil(t, respBody.Data.Item)

	actualMessage := respBody.Data.Item

	require.EqualValues(t, payload.SenderId, actualMessage.SenderId)
	require.EqualValues(t, payload.ReceiverId, actualMessage.ReceiverId)
	require.EqualValues(t, payload.Content, actualMessage.Content)
	require.EqualValues(t, payload.Type, actualMessage.Type)
	require.NotEmpty(t, actualMessage.Id)
	require.NotEmpty(t, actualMessage.CreatedAt)

	require.Len(t, actualMessage.Attachments, len(payload.Attachments))
	for i, att := range actualMessage.Attachments {
		require.NotEmpty(t, att.Id)
		require.EqualValues(t, payload.Attachments[i].AssetId, att.AssetId)
		require.EqualValues(t, payload.Attachments[i].Position, att.Position)
		require.EqualValues(t, payload.Attachments[i].Type, att.Type)
	}

	brokers := []string{helper.BrokerAddr}
	r := kk.NewReader(kk.ReaderConfig{
		Brokers:     brokers,
		Topic:       string(kafka.MESSAGE_RESOURCE_CREATED_TOPIC),
		StartOffset: kk.FirstOffset,
		GroupID:     KAFKA_GROUP_ID,
	})

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var m kk.Message
	for {
		m, err = r.ReadMessage(timeoutCtx)
		if err == nil {
			// check the received message is for the created message
			require.EqualValues(t, string(m.Key), fmt.Sprintf("%d", actualMessage.Id))
			require.EqualValues(t, string(m.Topic), kafka.MESSAGE_RESOURCE_CREATED_TOPIC)
			require.NotEmpty(t, m.Value)
			var receivedMessageEvent kafka.MessageResourceCreatedEvent
			err = json.Unmarshal(m.Value, &receivedMessageEvent)
			require.NoError(t, err)

			receivedMessage := receivedMessageEvent.Message
			require.EqualValues(t, actualMessage.Id, receivedMessage.Id)
			require.EqualValues(t, actualMessage.SenderId, receivedMessage.SenderId)
			require.EqualValues(t, actualMessage.ReceiverId, receivedMessage.ReceiverId)
			require.EqualValues(t, actualMessage.Content, receivedMessage.Content)
			require.EqualValues(t, actualMessage.Type, receivedMessage.Type)
			require.EqualValues(t, actualMessage.CreatedAt, receivedMessage.CreatedAt)
			require.EqualValues(t, actualMessage.Attachments, receivedMessage.Attachments)
			break
		}

		// Check if timeout expired
		if timeoutCtx.Err() != nil {
			require.Fail(t, fmt.Sprintf("Timeout expired while waiting for message from topic %s", kafka.MESSAGE_RESOURCE_CREATED_TOPIC))
		}

		helper.Logger.Errorfln("Failed to read message from topic %s: %v", kafka.MESSAGE_RESOURCE_CREATED_TOPIC, err)
		<-time.After(1 * time.Second)
		helper.Logger.Infofln("Retrying to read message from topic %s", kafka.MESSAGE_RESOURCE_CREATED_TOPIC)
	}
	helper.Logger.Infofln("Received message from topic %s: %s", kafka.MESSAGE_RESOURCE_CREATED_TOPIC, string(m.Value))
}

func TestCreateMessageWithEmptyAttachmentShouldWork(t *testing.T) {
	helper := NewTestHelper()
	SetUp(helper, &SetUpOptions{
		DataFile: &database.DataFile{
			UserFile:         USERS__MESSAGE__CREATE_FILENAME,
			ConversationFile: CONVS__MESSAGE__CREATE_FILENAME,
			AssetFile:        ASSETS__MESSAGE__CREATE_FILENAME,
		},
	})
	defer TearDown(helper)

	token := GetInternalAccessToken(2)
	payload := api.MessagePostRequestBody{
		SenderId:   types.NewJsonInt64(4),
		ReceiverId: types.NewJsonInt64(2),
		Content:    "Hey! Check out these files.",
		Type:       model.MSG_IMAGE,

		Attachments: []api.AttachmentPostRequestBody{},
	}
	payloadJson, err := json.Marshal(payload)
	require.NoError(t, err)

	url := fmt.Sprintf("%s/messages", helper.MessageServiceURL)
	header := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", token)},
	}
	resp, err := Post(helper.Client, url, header, bytes.NewBuffer(payloadJson))
	require.NoError(t, err)
	require.EqualValues(t, http.StatusCreated, resp.StatusCode)

	var respBody types.Response[model.Message]
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)

	require.Empty(t, respBody.Error)
	require.NotEmpty(t, respBody.Data)
	require.Nil(t, respBody.Data.Page)
	require.NotNil(t, respBody.Data.Item)

	actualMessage := respBody.Data.Item

	require.EqualValues(t, payload.SenderId, actualMessage.SenderId)
	require.EqualValues(t, payload.ReceiverId, actualMessage.ReceiverId)
	require.EqualValues(t, payload.Content, actualMessage.Content)
	require.EqualValues(t, payload.Type, actualMessage.Type)
	require.NotEmpty(t, actualMessage.Id)
	require.NotEmpty(t, actualMessage.CreatedAt)

	require.Len(t, actualMessage.Attachments, len(payload.Attachments))
	for i, att := range actualMessage.Attachments {
		require.NotEmpty(t, att.Id)
		require.EqualValues(t, payload.Attachments[i].AssetId, att.AssetId)
		require.EqualValues(t, payload.Attachments[i].Position, att.Position)
		require.EqualValues(t, payload.Attachments[i].Type, att.Type)
	}
}
