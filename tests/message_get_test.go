package tests

import (
	h "dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type GetMessagesResponseBody struct {
	HasMore bool            `json:"has_more"`
	Data    []model.Message `json:"data"`
}

const (
	USERS__MESSAGE__GET_FILENAME = "chat_users__message__get_test.json"
	CONVS__MESSAGE__GET_FILENAME = "conversations__message__get_test.json"
	MSGS__MESSAGE__GET_FILENAME  = "messages__message__get_test.json"
)

func TestGetMessagesShouldWork(t *testing.T) {
	helper := NewTestHelper()
	SetUp(helper, &SetUpOptions{
		DataFile: &database.DataFile{
			UserFile:         USERS__MESSAGE__GET_FILENAME,
			ConversationFile: CONVS__MESSAGE__GET_FILENAME,
			MessageFile:      MSGS__MESSAGE__GET_FILENAME,
		},
	})
	defer TearDown(helper)

	var convId int64
	convId = 2 // make sure it's in preset data
	token := GetInternalAccessToken(2)

	presetMessages, err := helper.AdminDatabaseService.GetAllMessages()
	require.NoError(t, err)
	presetMessages = h.Filter(presetMessages, func(msg model.Message) bool {
		return msg.ReceiverId.Int64() == convId
	})
	// ID desc
	sort.Slice(presetMessages, func(i, j int) bool {
		return presetMessages[i].Id.Int64() > presetMessages[j].Id.Int64()
	})

	url := fmt.Sprintf("%s/conversations/%d/messages", helper.MessageServiceURL, convId)

	header := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", token)},
	}
	resp, err := Get(helper.Client, url, header)
	require.NoError(t, err)
	require.EqualValues(t, http.StatusOK, resp.StatusCode)

	var respBody types.Response[model.Message]
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)

	require.Empty(t, respBody.Error)
	require.NotEmpty(t, respBody.Data)
	require.Nil(t, respBody.Data.Item)
	require.NotNil(t, respBody.Data.Page)

	actualMessages := respBody.Data.Page.Items

	require.EqualValues(t, len(presetMessages), len(actualMessages))
	for expected, actual := range h.Zip(presetMessages, actualMessages) {
		require.EqualValues(t, expected.Id, actual.Id)
		require.EqualValues(t, expected.Content, actual.Content)
		require.EqualValues(t, expected.Type, actual.Type)
		require.EqualValues(t, expected.SenderId, actual.SenderId)
		require.EqualValues(t, expected.ReceiverId, actual.ReceiverId)
		require.EqualValues(t, expected.ReplyToMessageId, actual.ReplyToMessageId)

		require.EqualValues(t, len(expected.Attachments), len(actual.Attachments))
		for expAtt, actAtt := range h.Zip(expected.Attachments, actual.Attachments) {
			require.EqualValues(t, expAtt.Id, actAtt.Id)
			require.EqualValues(t, expAtt.AssetId, actAtt.AssetId)
			require.EqualValues(t, expAtt.MessageId, actAtt.MessageId)
			require.EqualValues(t, expAtt.Position, actAtt.Position)
			require.EqualValues(t, expAtt.Type, actAtt.Type)
		}
	}
}

func TestGetMessagesWithDifferentLimitsShouldWork(t *testing.T) {
	helper := NewTestHelper()
	SetUp(helper, &SetUpOptions{
		DataFile: &database.DataFile{
			UserFile:         USERS__MESSAGE__GET_FILENAME,
			ConversationFile: CONVS__MESSAGE__GET_FILENAME,
			MessageFile:      MSGS__MESSAGE__GET_FILENAME,
		},
	})
	defer TearDown(helper)

	var convId int64
	convId = 2 // make sure it's in preset data
	token := GetInternalAccessToken(2)

	presetMessages, err := helper.AdminDatabaseService.GetAllMessages()
	require.NoError(t, err)
	presetMessages = h.Filter(presetMessages, func(msg model.Message) bool {
		return msg.ReceiverId.Int64() == convId
	})
	// ID desc
	sort.Slice(presetMessages, func(i, j int) bool {
		return presetMessages[i].Id.Int64() > presetMessages[j].Id.Int64()
	})
	presetMsgLen := len(presetMessages)

	testcases := []struct {
		message          string
		limit            string
		expectedMessages []model.Message
		hasMore          bool
		isSuccess        bool
	}{
		{
			message:          "limit equal to total",
			limit:            strconv.Itoa(presetMsgLen),
			expectedMessages: presetMessages[:int64(presetMsgLen)],
			hasMore:          false,
			isSuccess:        true,
		},
		{
			message:          "limit less than total",
			limit:            strconv.Itoa(presetMsgLen - 1),
			expectedMessages: presetMessages[:int64(presetMsgLen)-1],
			hasMore:          true,
			isSuccess:        true,
		},
		{
			message:          "limit more than total",
			limit:            strconv.Itoa(presetMsgLen + 1),
			expectedMessages: presetMessages[:int64(presetMsgLen)],
			hasMore:          false,
			isSuccess:        true,
		},
		{
			message:   "limit is negative",
			limit:     "-1",
			isSuccess: false,
		},
		{
			message:          "limit is zero",
			limit:            "0",
			expectedMessages: []model.Message{},
			hasMore:          true,
			isSuccess:        true,
		},
		{
			message:   "limit is not a number",
			limit:     "abc",
			isSuccess: false,
		},
	}

	for _, tc := range testcases {
		t.Run(
			fmt.Sprintf("limit: %s, hasMore: %t, expectedMesssages: %#v, message: %s, isSuccess: %t",
				tc.limit, tc.hasMore, tc.expectedMessages, tc.message, tc.isSuccess),
			func(t *testing.T) {
				url := fmt.Sprintf("%s/conversations/%d/messages?limit=%s", helper.MessageServiceURL, convId, tc.limit)
				header := http.Header{
					"Content-Type":  {"application/json"},
					"Authorization": {fmt.Sprintf("Bearer %s", token)},
				}
				resp, err := Get(helper.Client, url, header)
				require.NoError(t, err)

				var respBody types.Response[model.Message]
				err = json.NewDecoder(resp.Body).Decode(&respBody)
				require.NoError(t, err)
				if !tc.isSuccess {
					require.NotEmpty(t, respBody.Error)
					require.Empty(t, respBody.Data)
					return
				}

				require.Empty(t, respBody.Error)
				require.NotEmpty(t, respBody.Data)
				require.Nil(t, respBody.Data.Item)
				require.NotNil(t, respBody.Data.Page)

				require.EqualValues(t, tc.hasMore, respBody.Data.Page.HasMore)
				require.EqualValues(t, len(tc.expectedMessages), len(respBody.Data.Page.Items))

				actualMessages := respBody.Data.Page.Items
				for expected, actual := range h.Zip(tc.expectedMessages, actualMessages) {
					require.EqualValues(t, expected.Id, actual.Id)
					require.EqualValues(t, expected.Content, actual.Content)
					require.EqualValues(t, expected.Type, actual.Type)
					require.EqualValues(t, expected.SenderId, actual.SenderId)
					require.EqualValues(t, expected.ReceiverId, actual.ReceiverId)
					require.EqualValues(t, expected.ReplyToMessageId, actual.ReplyToMessageId)

					require.EqualValues(t, len(expected.Attachments), len(actual.Attachments))
					for expAtt, actAtt := range h.Zip(expected.Attachments, actual.Attachments) {
						require.EqualValues(t, expAtt.Id, actAtt.Id)
						require.EqualValues(t, expAtt.AssetId, actAtt.AssetId)
						require.EqualValues(t, expAtt.MessageId, actAtt.MessageId)
						require.EqualValues(t, expAtt.Position, actAtt.Position)
						require.EqualValues(t, expAtt.Type, actAtt.Type)
					}
				}
			})
	}
}

func TestGetMessagesWithDifferentIdOffsetsShouldWork(t *testing.T) {
	helper := NewTestHelper()
	SetUp(helper, &SetUpOptions{
		DataFile: &database.DataFile{
			UserFile:         USERS__MESSAGE__GET_FILENAME,
			ConversationFile: CONVS__MESSAGE__GET_FILENAME,
			MessageFile:      MSGS__MESSAGE__GET_FILENAME,
		},
	})
	defer TearDown(helper)

	var convId int64
	convId = 2 // make sure it's in preset data
	token := GetInternalAccessToken(2)

	presetMessages, err := helper.AdminDatabaseService.GetAllMessages()
	require.NoError(t, err)
	presetMessages = h.Filter(presetMessages, func(msg model.Message) bool {
		return msg.ReceiverId.Int64() == convId
	})
	// ID desc
	sort.Slice(presetMessages, func(i, j int) bool {
		return presetMessages[i].Id.Int64() > presetMessages[j].Id.Int64()
	})

	// max to min
	ids := h.Map(presetMessages, func(msg model.Message) int64 {
		return msg.Id.Int64()
	})

	ids = append(ids, ids[len(ids)-1]-1) // less than min
	ids = append(ids, ids[0]+1)          // more than max

	testcases := []struct {
		message   string
		after     string
		isSuccess bool
	}{
		{
			message:   "after is not a number",
			after:     "abc",
			isSuccess: false,
		},
		{
			message:   "after is negative",
			after:     "-1",
			isSuccess: false,
		},
	}

	for _, id := range ids {
		testcases = append(testcases, struct {
			message   string
			after     string
			isSuccess bool
		}{
			message:   fmt.Sprintf("after is %d", id),
			after:     fmt.Sprintf("%d", id),
			isSuccess: true,
		})
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf(
			"after: %s, isSuccess: %t, message: %s",
			tc.after, tc.isSuccess, tc.message,
		), func(t *testing.T) {
			url := fmt.Sprintf("%s/conversations/%d/messages?after=%s", helper.MessageServiceURL, convId, tc.after)
			header := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", token)},
			}
			resp, err := Get(helper.Client, url, header)
			require.NoError(t, err)

			var respBody types.Response[model.Message]
			err = json.NewDecoder(resp.Body).Decode(&respBody)
			require.NoError(t, err)

			if !tc.isSuccess {
				require.NotEmpty(t, respBody.Error)
				require.Empty(t, respBody.Data)
				return
			}

			expectedMessages := h.Filter(presetMessages, func(msg model.Message) bool {
				after, err := strconv.ParseInt(tc.after, 10, 64)
				require.NoError(t, err)
				return msg.Id.Int64() < after // ID desc
			})

			require.EqualValues(t, http.StatusOK, resp.StatusCode)

			require.Empty(t, respBody.Error)
			require.NotEmpty(t, respBody.Data)
			require.Nil(t, respBody.Data.Item)
			require.NotNil(t, respBody.Data.Page)
			require.EqualValues(t, false, respBody.Data.Page.HasMore)

			actualMessages := respBody.Data.Page.Items
			require.EqualValues(t, len(expectedMessages), len(actualMessages))
			for expected, actual := range h.Zip(expectedMessages, actualMessages) {
				require.EqualValues(t, expected.Id, actual.Id)
				require.EqualValues(t, expected.Content, actual.Content)
				require.EqualValues(t, expected.Type, actual.Type)
				require.EqualValues(t, expected.SenderId, actual.SenderId)
				require.EqualValues(t, expected.ReceiverId, actual.ReceiverId)
				require.EqualValues(t, expected.ReplyToMessageId, actual.ReplyToMessageId)

				require.EqualValues(t, len(expected.Attachments), len(actual.Attachments))
				for expAtt, actAtt := range h.Zip(expected.Attachments, actual.Attachments) {
					require.EqualValues(t, expAtt.Id, actAtt.Id)
					require.EqualValues(t, expAtt.AssetId, actAtt.AssetId)
					require.EqualValues(t, expAtt.MessageId, actAtt.MessageId)
					require.EqualValues(t, expAtt.Position, actAtt.Position)
					require.EqualValues(t, expAtt.Type, actAtt.Type)
				}
			}
		})
	}
}

func TestGetMessagesWithAllOptsShouldWork(t *testing.T) {
	helper := NewTestHelper()
	SetUp(helper, &SetUpOptions{
		DataFile: &database.DataFile{
			UserFile:         USERS__MESSAGE__GET_FILENAME,
			ConversationFile: CONVS__MESSAGE__GET_FILENAME,
			MessageFile:      MSGS__MESSAGE__GET_FILENAME,
		},
	})
	defer TearDown(helper)

	var convId int64
	convId = 2 // make sure it's in preset data
	token := GetInternalAccessToken(2)

	presetMessages, err := helper.AdminDatabaseService.GetAllMessages()
	require.NoError(t, err)
	presetMessages = h.Filter(presetMessages, func(msg model.Message) bool {
		return msg.ReceiverId.Int64() == convId
	})
	// ID desc
	sort.Slice(presetMessages, func(i, j int) bool {
		return presetMessages[i].Id.Int64() > presetMessages[j].Id.Int64()
	})

	for range 20 {
		after := h.RandRange(int(presetMessages[len(presetMessages)-1].Id.Int64()), int(presetMessages[0].Id.Int64()))
		limit := h.RandRange(1, len(presetMessages))

		expectedMessages := make([]model.Message, len(presetMessages))
		copy(expectedMessages, presetMessages)
		expectedMessages = h.Filter(expectedMessages, func(m model.Message) bool {
			return m.Id.Int64() < int64(after)
		})
		sort.Slice(expectedMessages, func(i, j int) bool {
			return expectedMessages[i].Id.Int64() > expectedMessages[j].Id.Int64()
		})

		prevLen := len(expectedMessages)
		expectedMessages = expectedMessages[:int(math.Min(float64(prevLen), float64(limit)))]
		hasMore := false
		if len(expectedMessages) < prevLen {
			hasMore = true
		}

		t.Run(fmt.Sprintf(
			"after: %d, limit: %d, hasMore: %t",
			after, limit, hasMore,
		),
			func(t *testing.T) {
				url := fmt.Sprintf("%s/conversations/%d/messages?after=%d&limit=%d", helper.MessageServiceURL, convId, after, limit)
				header := http.Header{
					"Content-Type":  {"application/json"},
					"Authorization": {fmt.Sprintf("Bearer %s", token)},
				}
				resp, err := Get(helper.Client, url, header)
				require.NoError(t, err)
				require.EqualValues(t, http.StatusOK, resp.StatusCode)

				var respBody types.Response[model.Message]
				err = json.NewDecoder(resp.Body).Decode(&respBody)
				require.NoError(t, err)

				require.Empty(t, respBody.Error)
				require.NotEmpty(t, respBody.Data)
				require.Nil(t, respBody.Data.Item)
				require.NotNil(t, respBody.Data.Page)

				require.EqualValues(t, hasMore, respBody.Data.Page.HasMore)

				actualMessages := respBody.Data.Page.Items
				require.EqualValues(t, len(expectedMessages), len(actualMessages))
				for expected, actual := range h.Zip(expectedMessages, actualMessages) {
					require.EqualValues(t, expected.Id, actual.Id)
					require.EqualValues(t, expected.Content, actual.Content)
					require.EqualValues(t, expected.Type, actual.Type)
					require.EqualValues(t, expected.SenderId, actual.SenderId)
					require.EqualValues(t, expected.ReceiverId, actual.ReceiverId)
					require.EqualValues(t, expected.ReplyToMessageId, actual.ReplyToMessageId)

					require.EqualValues(t, len(expected.Attachments), len(actual.Attachments))
					for expAtt, actAtt := range h.Zip(expected.Attachments, actual.Attachments) {
						require.EqualValues(t, expAtt.Id, actAtt.Id)
						require.EqualValues(t, expAtt.AssetId, actAtt.AssetId)
						require.EqualValues(t, expAtt.MessageId, actAtt.MessageId)
						require.EqualValues(t, expAtt.Position, actAtt.Position)
						require.EqualValues(t, expAtt.Type, actAtt.Type)
					}
				}
			})
	}
}
