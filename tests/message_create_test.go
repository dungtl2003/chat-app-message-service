package tests

import (
	"bytes"
	"dungtl2003/chat-app-message-service/internal/api"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

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
