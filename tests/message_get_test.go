package tests

import (
	h "dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"fmt"
	"math"
	"net/http"
	"sort"
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
	helper := NewHelper()
	err := helper.CreateTemporaryData(database.DataFile{
		UserFile:         USERS__MESSAGE__GET_FILENAME,
		ConversationFile: CONVS__MESSAGE__GET_FILENAME,
		MessageFile:      MSGS__MESSAGE__GET_FILENAME,
	})
	require.NoError(t, err)
	defer func() {
		err := helper.ClearAllData()
		require.NoError(t, err)
		err = helper.Db.Close()
		require.NoError(t, err)
	}()

	var convId int64
	convId = 2 // make sure it's in preset data

	presetMessages, err := helper.Db.GetAllMessages()
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
		"Authorization": {fmt.Sprintf("Bearer %s", JWT_USER_ID_2)},
	}
	resp, err := helper.Client.Get(url, header)
	require.NoError(t, err)
	respBody, err := helper.Client.ReadResponse(resp)
	require.NoError(t, err)
	require.EqualValues(t, http.StatusOK, resp.StatusCode)

	var getMsgRespBody GetMessagesResponseBody
	err = h.ParseAsJson(respBody, &getMsgRespBody)
	require.NoError(t, err)
	respMessages := getMsgRespBody.Data
	require.EqualValues(t, len(presetMessages), len(respMessages))
	for expected, actual := range h.Zip(presetMessages, respMessages) {
		require.EqualValues(t, expected.Id, actual.Id)
		require.EqualValues(t, expected.Content, actual.Content)
		require.EqualValues(t, expected.Type, actual.Type)
		require.EqualValues(t, expected.SenderId, actual.SenderId)
		require.EqualValues(t, expected.ReceiverId, actual.ReceiverId)
	}
}

func TestGetMessagesWithDifferentLimitsShouldWork(t *testing.T) {
	helper := NewHelper()
	err := helper.CreateTemporaryData(database.DataFile{
		UserFile:         USERS__MESSAGE__GET_FILENAME,
		ConversationFile: CONVS__MESSAGE__GET_FILENAME,
		MessageFile:      MSGS__MESSAGE__GET_FILENAME,
	})
	require.NoError(t, err)
	defer func() {
		err := helper.ClearAllData()
		require.NoError(t, err)
		err = helper.Db.Close()
		require.NoError(t, err)
	}()

	var convId int64
	convId = 2 // make sure it's in preset data

	presetMessages, err := helper.Db.GetAllMessages()
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
		limit            int64
		expectedMessages []model.Message
		hasMore          bool
	}{
		{
			limit:            int64(presetMsgLen),
			expectedMessages: presetMessages[:int64(presetMsgLen)],
			hasMore:          false,
		},
		{
			limit:            int64(presetMsgLen) - 1,
			expectedMessages: presetMessages[:int64(presetMsgLen)-1],
			hasMore:          true,
		},
		{
			limit:            int64(presetMsgLen) + 1,
			expectedMessages: presetMessages[:int64(presetMsgLen)],
			hasMore:          false,
		},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("limit: %d, hasMore: %t, expectedMesssages: %#v", tc.limit, tc.hasMore, tc.expectedMessages), func(t *testing.T) {
			url := fmt.Sprintf("%s/conversations/%d/messages?limit=%d", helper.MessageServiceURL, convId, tc.limit)
			header := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", JWT_USER_ID_2)},
			}
			resp, err := helper.Client.Get(url, header)
			require.NoError(t, err)
			respBody, err := helper.Client.ReadResponse(resp)
			require.NoError(t, err)
			require.EqualValues(t, http.StatusOK, resp.StatusCode)

			var getMsgRespBody GetMessagesResponseBody
			err = h.ParseAsJson(respBody, &getMsgRespBody)
			require.NoError(t, err)
			require.EqualValues(t, tc.hasMore, getMsgRespBody.HasMore)
			respMessages := getMsgRespBody.Data
			require.EqualValues(t, len(tc.expectedMessages), len(respMessages))
			for expected, actual := range h.Zip(tc.expectedMessages, respMessages) {
				require.EqualValues(t, expected.Id, actual.Id)
				require.EqualValues(t, expected.Content, actual.Content)
				require.EqualValues(t, expected.Type, actual.Type)
				require.EqualValues(t, expected.SenderId, actual.SenderId)
				require.EqualValues(t, expected.ReceiverId, actual.ReceiverId)
			}
		})
	}
}

func TestGetMessagesWithDifferentOrdersShouldWork(t *testing.T) {
	helper := NewHelper()
	err := helper.CreateTemporaryData(database.DataFile{
		UserFile:         USERS__MESSAGE__GET_FILENAME,
		ConversationFile: CONVS__MESSAGE__GET_FILENAME,
		MessageFile:      MSGS__MESSAGE__GET_FILENAME,
	})
	require.NoError(t, err)
	defer func() {
		err := helper.ClearAllData()
		require.NoError(t, err)
		err = helper.Db.Close()
		require.NoError(t, err)
	}()

	var convId int64
	convId = 2 // make sure it's in preset data

	presetMessages, err := helper.Db.GetAllMessages()
	require.NoError(t, err)
	presetMessages = h.Filter(presetMessages, func(msg model.Message) bool {
		return msg.ReceiverId.Int64() == convId
	})

	testcases := []struct {
		lessFunc func(messages []model.Message) func(i, j int) bool
		orderBy  string
	}{
		{
			lessFunc: func(messages []model.Message) func(i, j int) bool {
				return func(i, j int) bool {
					return messages[i].Id.Int64() < messages[j].Id.Int64()
				}
			},
			orderBy: "id:asc",
		},
		{
			lessFunc: func(messages []model.Message) func(i, j int) bool {
				return func(i, j int) bool {
					return messages[i].Id.Int64() > messages[j].Id.Int64()
				}
			},
			orderBy: "id:desc",
		},
		{
			lessFunc: func(messages []model.Message) func(i, j int) bool {
				return func(i, j int) bool {
					return messages[i].Content < messages[j].Content
				}
			},
			orderBy: "content:asc",
		},
		{
			lessFunc: func(messages []model.Message) func(i, j int) bool {
				return func(i, j int) bool {
					return messages[i].Content > messages[j].Content
				}
			},
			orderBy: "content:desc",
		},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("orderBy: %s", tc.orderBy), func(t *testing.T) {
			url := fmt.Sprintf("%s/conversations/%d/messages?order_by=%s", helper.MessageServiceURL, convId, tc.orderBy)
			header := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", JWT_USER_ID_2)},
			}
			resp, err := helper.Client.Get(url, header)
			require.NoError(t, err)
			respBody, err := helper.Client.ReadResponse(resp)
			require.NoError(t, err)
			require.EqualValues(t, http.StatusOK, resp.StatusCode)

			var getMsgRespBody GetMessagesResponseBody
			err = h.ParseAsJson(respBody, &getMsgRespBody)
			require.NoError(t, err)
			require.EqualValues(t, false, getMsgRespBody.HasMore)
			respMessages := getMsgRespBody.Data
			require.EqualValues(t, len(presetMessages), len(respMessages))

			expectedMessages := make([]model.Message, len(presetMessages))
			copy(expectedMessages, presetMessages)
			sort.Slice(expectedMessages, tc.lessFunc(expectedMessages))
			for expected, actual := range h.Zip(expectedMessages, respMessages) {
				require.EqualValues(t, expected.Id, actual.Id)
				require.EqualValues(t, expected.Content, actual.Content)
				require.EqualValues(t, expected.Type, actual.Type)
				require.EqualValues(t, expected.SenderId, actual.SenderId)
				require.EqualValues(t, expected.ReceiverId, actual.ReceiverId)
			}
		})
	}
}

func TestGetMessagesWithDifferentIdOffsetsShouldWork(t *testing.T) {
	helper := NewHelper()
	err := helper.CreateTemporaryData(database.DataFile{
		UserFile:         USERS__MESSAGE__GET_FILENAME,
		ConversationFile: CONVS__MESSAGE__GET_FILENAME,
		MessageFile:      MSGS__MESSAGE__GET_FILENAME,
	})
	require.NoError(t, err)
	defer func() {
		err := helper.ClearAllData()
		require.NoError(t, err)
		err = helper.Db.Close()
		require.NoError(t, err)
	}()

	var convId int64
	convId = 2 // make sure it's in preset data

	presetMessages, err := helper.Db.GetAllMessages()
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

	for _, after := range ids {
		t.Run(fmt.Sprintf("after: %d", after), func(t *testing.T) {
			expectedMessages := h.Filter(presetMessages, func(msg model.Message) bool {
				return msg.Id.Int64() > after
			})

			url := fmt.Sprintf("%s/conversations/%d/messages?after=%d", helper.MessageServiceURL, convId, after)
			header := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", JWT_USER_ID_2)},
			}
			resp, err := helper.Client.Get(url, header)
			require.NoError(t, err)
			respBody, err := helper.Client.ReadResponse(resp)
			require.NoError(t, err)
			require.EqualValues(t, http.StatusOK, resp.StatusCode)

			var getMsgRespBody GetMessagesResponseBody
			err = h.ParseAsJson(respBody, &getMsgRespBody)
			require.NoError(t, err)
			respMessages := getMsgRespBody.Data
			require.EqualValues(t, false, getMsgRespBody.HasMore)
			require.EqualValues(t, len(expectedMessages), len(respMessages))

			for expected, actual := range h.Zip(expectedMessages, respMessages) {
				require.EqualValues(t, expected.Id, actual.Id)
				require.EqualValues(t, expected.Content, actual.Content)
				require.EqualValues(t, expected.Type, actual.Type)
				require.EqualValues(t, expected.SenderId, actual.SenderId)
				require.EqualValues(t, expected.ReceiverId, actual.ReceiverId)
			}
		})
	}
}

func TestGetMessagesWithAllOptsShouldWork(t *testing.T) {
	helper := NewHelper()
	err := helper.CreateTemporaryData(database.DataFile{
		UserFile:         USERS__MESSAGE__GET_FILENAME,
		ConversationFile: CONVS__MESSAGE__GET_FILENAME,
		MessageFile:      MSGS__MESSAGE__GET_FILENAME,
	})
	require.NoError(t, err)
	defer func() {
		err := helper.ClearAllData()
		require.NoError(t, err)
		err = helper.Db.Close()
		require.NoError(t, err)
	}()

	var convId int64
	convId = 2 // make sure it's in preset data

	presetMessages, err := helper.Db.GetAllMessages()
	require.NoError(t, err)
	presetMessages = h.Filter(presetMessages, func(msg model.Message) bool {
		return msg.ReceiverId.Int64() == convId
	})
	// ID desc
	sort.Slice(presetMessages, func(i, j int) bool {
		return presetMessages[i].Id.Int64() > presetMessages[j].Id.Int64()
	})

	orders := []string{"asc", "desc"}
	for range 20 {
		// for simplicity, we will only order ID field
		randOrderIdx := h.RandRange(0, len(orders)-1)
		randOrder := orders[randOrderIdx]

		after := h.RandRange(int(presetMessages[len(presetMessages)-1].Id.Int64()), int(presetMessages[0].Id.Int64()))
		limit := h.RandRange(1, len(presetMessages))
		orderBy := fmt.Sprintf("id:%s", randOrder)

		expectedMessages := make([]model.Message, len(presetMessages))
		copy(expectedMessages, presetMessages)
		expectedMessages = h.Filter(expectedMessages, func(m model.Message) bool {
			return m.Id.Int64() > int64(after)
		})
		sort.Slice(expectedMessages, func(i, j int) bool {
			if randOrder == "asc" {
				return expectedMessages[i].Id.Int64() < expectedMessages[j].Id.Int64()
			} else {
				return expectedMessages[i].Id.Int64() > expectedMessages[j].Id.Int64()
			}
		})

		prevLen := len(expectedMessages)
		expectedMessages = expectedMessages[:int(math.Min(float64(prevLen), float64(limit)))]
		hasMore := false
		if len(expectedMessages) < prevLen {
			hasMore = true
		}

		t.Run(fmt.Sprintf("after: %d, limit: %d, orderBy: %s, hasMore: %t", after, limit, orderBy, hasMore), func(t *testing.T) {
			url := fmt.Sprintf("%s/conversations/%d/messages?after=%d&limit=%d&order_by=%s", helper.MessageServiceURL, convId, after, limit, orderBy)
			header := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", JWT_USER_ID_2)},
			}
			resp, err := helper.Client.Get(url, header)
			require.NoError(t, err)
			require.EqualValues(t, http.StatusOK, resp.StatusCode)
			respBody, err := helper.Client.ReadResponse(resp)
			require.NoError(t, err)

			var getMsgRespBody GetMessagesResponseBody
			err = h.ParseAsJson(respBody, &getMsgRespBody)
			require.NoError(t, err)
			helper.Logger.Debugfln("received body: %#v", getMsgRespBody)
			respMessages := getMsgRespBody.Data
			require.EqualValues(t, hasMore, getMsgRespBody.HasMore)
			require.EqualValues(t, len(expectedMessages), len(respMessages))

			for expected, actual := range h.Zip(expectedMessages, respMessages) {
				require.EqualValues(t, expected.Id, actual.Id)
				require.EqualValues(t, expected.Content, actual.Content)
				require.EqualValues(t, expected.Type, actual.Type)
				require.EqualValues(t, expected.SenderId, actual.SenderId)
				require.EqualValues(t, expected.ReceiverId, actual.ReceiverId)
			}
		})
	}
}

func TestGetMessagesWithInvalidOptsShouldFail(t *testing.T) {
	helper := NewHelper()
	err := helper.CreateTemporaryData(database.DataFile{
		UserFile:         USERS__MESSAGE__GET_FILENAME,
		ConversationFile: CONVS__MESSAGE__GET_FILENAME,
		MessageFile:      MSGS__MESSAGE__GET_FILENAME,
	})
	require.NoError(t, err)
	defer func() {
		err := helper.ClearAllData()
		require.NoError(t, err)
		err = helper.Db.Close()
		require.NoError(t, err)
	}()

	params := []string{"", "abc", "2", "2", "2", "2", "2", "2", "2"}

	queries := []string{
		"",
		"",
		"?after=abc",
		"?after=-1",
		"?limit=abc",
		"?limit=-1",
		"?order_by=abc",
		"?order_by=abc:asc",
		"?order_by=id:DESC",
	}

	for param, query := range h.Zip(params, queries) {
		t.Run(fmt.Sprintf("query: %s", query), func(t *testing.T) {
			url := fmt.Sprintf("%s/conversations/%s/messages%s", helper.MessageServiceURL, param, query)
			header := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", JWT_USER_ID_2)},
			}
			resp, err := helper.Client.Get(url, header)
			require.NoError(t, err)
			require.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}
