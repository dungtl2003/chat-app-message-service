package tests

import (
	"bytes"
	h "dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	USERS__MESSAGE__CREATE_FILENAME = "chat_users__message__create_test.json"
	CONVS__MESSAGE__CREATE_FILENAME = "conversations__message__create_test.json"
)

func TestCreateMessageShouldWork(t *testing.T) {
	helper := NewHelper()
	err := helper.CreateTemporaryData(database.DataFile{
		UserFile:         USERS__MESSAGE__CREATE_FILENAME,
		ConversationFile: CONVS__MESSAGE__CREATE_FILENAME,
	})
	require.NoError(t, err)
	defer func() {
		err := helper.ClearAllData()
		require.NoError(t, err)
		err = helper.Db.Close()
		require.NoError(t, err)
	}()

	receiverId := 2 // make sure it's in preset data
	senderId := 4
	content := "Hey! Check out these files."
	msgType := model.VIDEO
	thumbURLs := []string{
		"https://example.com/thumb1.jpg",
		"https://example.com/thumb2.jpg",
		"https://example.com/thumb3.jpg",
	}
	fileURLs := []string{
		"https://example.com/file1.jpg",
		"https://example.com/file2.jpg",
		"https://example.com/file3.jpg",
	}
	payload := fmt.Appendf(nil, `
		{
	  		"sender_id": "%d",
	  		"receiver_id": "%d",
	  		"content": "%s",
	  		"type": "%s",
	  		"attachments": [
				{
				  	"thumb_url": "%s",
				  	"file_url": "%s"
				},
				{
				  	"thumb_url": "%s",
				  	"file_url": "%s"
				},
				{
				  	"thumb_url": "%s",
				  	"file_url": "%s"
				}
	  		]
		}
	`, senderId, receiverId, content, msgType, thumbURLs[0], fileURLs[0], thumbURLs[1], fileURLs[1], thumbURLs[2], fileURLs[2])

	url := fmt.Sprintf("%s/messages", helper.MessageServiceURL)

	header := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", JWT_USER_ID_2)},
	}
	resp, err := helper.Client.Post(url, header, bytes.NewBuffer(payload))
	require.NoError(t, err)
	respBody, err := helper.Client.ReadResponse(resp)
	require.NoError(t, err)
	require.EqualValues(t, http.StatusCreated, resp.StatusCode)

	var postMsgRespBody model.Message
	err = h.ParseAsJson(respBody, &postMsgRespBody)
	require.NoError(t, err)

	require.EqualValues(t, senderId, postMsgRespBody.SenderId.Int64())
	require.EqualValues(t, receiverId, postMsgRespBody.ReceiverId.Int64())
	require.EqualValues(t, content, postMsgRespBody.Content)
	require.EqualValues(t, msgType, postMsgRespBody.Type)
	require.EqualValues(t, 3, len(postMsgRespBody.Attachments))

	idx := 0
	for thumbURL, fileURL := range h.Zip(thumbURLs, fileURLs) {
		require.EqualValues(t, thumbURL, postMsgRespBody.Attachments[idx].ThumbURL)
		require.EqualValues(t, fileURL, postMsgRespBody.Attachments[idx].FileURL)
		idx++
	}
}
