package tests

import (
	"bytes"
	"context"
	"dungtl2003/chat-app-message-service/internal/api"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/server"
	"dungtl2003/chat-app-message-service/internal/services/conversation"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/kafka"
	"dungtl2003/chat-app-message-service/internal/services/user"
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	gokafka "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

const (
	USERS__MSG__CREATE_FILENAME  = "chat_users__message__create_test.json"
	CONVS__MSG__CREATE_FILENAME  = "conversations__message__create_test.json"
	ASSETS__MSG__CREATE_FILENAME = "assets__message__create_test.json"
)

func TestMessageCreateFlowShouldWork(t *testing.T) {
	helper := NewTestHelper()
	SetUp(helper, &SetUpOptions{
		DataFile: &database.DataFile{
			UserFile:         USERS__MSG__CREATE_FILENAME,
			ConversationFile: CONVS__MSG__CREATE_FILENAME,
			AssetFile:        ASSETS__MSG__CREATE_FILENAME,
		},
		ServerOptions: &server.MessageServerOptions{
			UserService: &user.MockUserService{
				MockGetUsers: func(req *user.GetUsersRequest) (*user.GetUsersResponse, error) {
					users, err := helper.AdminDatabaseService.GetUsersByIds(t.Context(), req.UserIDs)
					require.NoError(t, err)

					userMap := make(map[int64]model.ChatUser)
					for _, u := range users {
						if u.AvatarId.Valid {
							u.AvatarURL = types.NewJsonNullString(fmt.Sprintf("https://fake.media.service/assets/%d", u.AvatarId.Int64))
						}
						userMap[u.Id.Int64()] = u
					}

					return &user.GetUsersResponse{
						UserMap: userMap,
					}, nil
				},
			},
			ConversationService: &conversation.MockConversationService{
				MockIsParticipantFunc: func(conversationID, participantID int64, internalToken string) (bool, error) {
					return true, nil
				},
				MockBatchGetParticipants: func(req *conversation.BatchGetParticipantsRequest) (*conversation.BatchGetParticipantsResponse, error) {
					participants, err := helper.AdminDatabaseService.GetParticipantsByConversationIdAndUserIds(t.Context(), req.ConversationID, req.UserIDs)
					require.NoError(t, err)

					participantMap := make(map[int64]model.Participant)
					for _, p := range participants {
						participantMap[p.UserId.Int64()] = p
					}

					return &conversation.BatchGetParticipantsResponse{
						ParticipantMap: participantMap,
					}, nil
				},
			},
		},
	})
	defer TearDown(helper)

	// Load assets to pick one
	assetsFile := fmt.Sprintf("%s/%s", helper.DataFileDir, ASSETS__MSG__CREATE_FILENAME)
	assetsData, err := os.ReadFile(assetsFile)
	require.NoError(t, err)
	var assets []model.Asset
	err = json.Unmarshal(assetsData, &assets)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(assets), 2)
	firstAsset := assets[0]
	secondAsset := assets[1]

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
		SenderId:       types.NewJsonInt64(senderId).ToPtr(),
		ReceiverId:     types.NewJsonInt64(receiverId).ToPtr(),
		Content:        "Hello from user 4 to user 2",
		Type:           model.MSG_TEXT,
		IdempotencyKey: "unique-key-12345",
		Attachments: []model.Attachment{
			{
				AssetId:  types.NewJsonInt64(firstAsset.Id.Int64()),
				Position: 1,
				Type:     model.ATT_IMAGE,
				Asset:    &firstAsset,
			},
			{
				AssetId:  types.NewJsonInt64(secondAsset.Id.Int64()),
				Position: 2,
				Type:     model.ATT_IMAGE,
				Asset:    &secondAsset,
			},
		},
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
	require.NotEmpty(t, respMsg.Attachments)
	require.Len(t, respMsg.Attachments, 2)

	// Create map to check for existence easily regardless of order
	respAttachmentMap := make(map[int64]model.Attachment)
	for _, att := range respMsg.Attachments {
		respAttachmentMap[att.AssetId.Int64()] = att
	}

	require.Contains(t, respAttachmentMap, firstAsset.Id.Int64())
	require.Contains(t, respAttachmentMap, secondAsset.Id.Int64())
	require.Equal(t, 1, respAttachmentMap[firstAsset.Id.Int64()].Position)
	require.Equal(t, 2, respAttachmentMap[secondAsset.Id.Int64()].Position)

	// Verify Asset object
	require.NotNil(t, respAttachmentMap[firstAsset.Id.Int64()].Asset)
	require.Equal(t, firstAsset.Id.Int64(), respAttachmentMap[firstAsset.Id.Int64()].Asset.Id.Int64())
	require.NotNil(t, respAttachmentMap[secondAsset.Id.Int64()].Asset)
	require.Equal(t, secondAsset.Id.Int64(), respAttachmentMap[secondAsset.Id.Int64()].Asset.Id.Int64())

	// Verify References
	require.NotNil(t, respBody.References)
	usersRef, ok := respBody.References["users"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, usersRef, strconv.FormatInt(senderId, 10))

	_, ok = respBody.References["participants"].(map[string]any)
	require.True(t, ok)

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
	require.NotEmpty(t, event.Message.Attachments)
	require.Len(t, event.Message.Attachments, 2)

	// Verify attachments in Kafka event
	eventAttachmentMap := make(map[int64]model.Attachment)
	for _, att := range event.Message.Attachments {
		eventAttachmentMap[att.AssetId.Int64()] = att
	}
	require.Contains(t, eventAttachmentMap, firstAsset.Id.Int64())
	require.Contains(t, eventAttachmentMap, secondAsset.Id.Int64())

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
