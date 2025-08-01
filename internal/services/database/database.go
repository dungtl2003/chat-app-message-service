package database

import (
	"database/sql"
	"dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services"
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	_ "github.com/lib/pq" // postgresql driver support
)

type DataFile struct {
	UserFile         string
	ConversationFile string
	ParticipantFile  string
	MessageFile      string
}

var (
	ErrDatabaseError = fmt.Errorf("database error")
)

type DatabaseService struct {
	client *sql.DB
	logger *logging.LoggerWrapper
	status services.ServiceStatus
}

func (d *DatabaseService) Name() string {
	return "Database Service"
}

func (d *DatabaseService) Status() services.ServiceStatus {
	if d.status != services.STOPPED {
		// check if the database connection is still alive
		if err := d.client.Ping(); err != nil {
			d.logger.Errorfln("[%s] Database connection is not alive: %v", d.Name(), err)
			d.status = services.ERROR
		} else {
			d.status = services.READY
		}
	}

	return d.status
}

// New creates a new database connection. The function returns a database connection and an error.
func New(url string, logger *logging.LoggerWrapper) (*DatabaseService, error) {
	client, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	d := &DatabaseService{
		client: client,
		logger: logger,
		status: services.READY,
	}

	d.logger.Infofln("[%s] Database connection created", d.Name())
	d.logger.Infofln("[%s] Running", d.Name())
	return d, nil

}

// Close closes the database connection. The function returns an error.
func (d *DatabaseService) Close() error {
	if d.Status() == services.STOPPED {
		d.logger.Errorfln("[%s] Database connection is already closed", d.Name())
		return nil
	}

	err := d.client.Close()
	if err != nil {
		d.logger.Errorfln("[%s] Failed to close database connection: %v", d.Name(), err)
		d.status = services.ERROR
	} else {
		d.logger.Infofln("[%s] Database connection closed", d.Name())
		d.status = services.STOPPED
	}

	return err
}

// CreateMessage create a new message. It will return the created message, status
// and error if occurs.
// The status code is 200 if the operation is done successfully.
// The status code is 500 database error occurs.
func (d *DatabaseService) CreateMessage(message model.Message) (*model.Message, int, error) {
	if d.Status() != services.READY {
		d.logger.Errorfln("[%s] Database is not ready", d.Name())
		return nil, http.StatusInternalServerError, ErrDatabaseError
	}

	tx, err := d.client.Begin()
	if err != nil {
		d.logger.Errorfln("client.Begin(): %v", err)
		return nil, http.StatusInternalServerError, ErrDatabaseError
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `
		INSERT INTO message.message (
			id, content, type, created_at, updated_at, deleted_at, sender_id, receiver_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		);`
	args := []any{message.Id, message.Content, message.Type, message.CreatedAt, message.UpdatedAt, message.DeletedAt, message.SenderId, message.ReceiverId}
	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	_, err = tx.Exec(query, args...)
	if err != nil {
		d.logger.Errorfln("tx.Exec(): %v", err)
		return nil, http.StatusInternalServerError, ErrDatabaseError
	}

	fields := reflect.Indirect(reflect.ValueOf(model.Attachment{})).Type().NumField() + 1 // count message ID as well
	args = make([]any, len(message.Attachments)*fields)
	argsCount := 1
	values := make([]string, len(message.Attachments))
	for i, attachment := range message.Attachments {
		values[i] = fmt.Sprintf(`($%d, $%d, $%d, $%d, $%d)`, argsCount, argsCount+1, argsCount+2, argsCount+3, argsCount+4)
		args[argsCount-1] = attachment.Id
		args[argsCount] = attachment.ThumbURL
		args[argsCount+1] = attachment.FileURL
		args[argsCount+2] = attachment.DeletedAt
		args[argsCount+3] = message.Id
		argsCount += 5
	}
	query = fmt.Sprintf(`
		INSERT INTO message.attachment (
			id, thumb_url, file_url, deleted_at, message_id
		) VALUES %s;
	`, strings.Join(values, ", "))
	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	_, err = tx.Exec(query, args...)
	if err != nil {
		d.logger.Errorfln("tx.Exec(): %v", err)
		return nil, http.StatusInternalServerError, ErrDatabaseError
	}

	query = `
		SELECT 
			m.id, m.content, m.type, m.created_at, m.updated_at, m.deleted_at, m.sender_id, m.receiver_id,
		    COALESCE(
            	json_agg(
                	json_build_object(
                    	'id', a.id,
						'thumb_url', a.thumb_url,
						'file_url', a.file_url,
						'deleted_at', a.deleted_at
                	)
            	) FILTER (WHERE a.id IS NOT NULL), '[]'
        	) AS attachments
    	FROM message.message m
    	LEFT JOIN message.attachment a ON m.id = a.message_id
    	WHERE m.id = $1
    	GROUP BY m.id;
	`
	args = []any{message.Id}
	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	var msg model.Message
	var rawAttachments json.RawMessage
	messageRow := tx.QueryRow(query, args...)
	err = messageRow.Scan(&msg.Id, &msg.Content, &msg.Type, &msg.CreatedAt, &msg.UpdatedAt, &msg.DeletedAt, &msg.SenderId, &msg.ReceiverId, &rawAttachments)
	if err != nil {
		d.logger.Errorfln("messageRow.Scan(): %v", err)
		return nil, http.StatusInternalServerError, ErrDatabaseError
	}

	err = json.Unmarshal(rawAttachments, &msg.Attachments)
	if err != nil {
		d.logger.Errorfln("json.Unmarshal(): %v", err)
		return nil, http.StatusInternalServerError, ErrDatabaseError
	}

	err = tx.Commit()
	if err != nil {
		d.logger.Errorfln("tx.Commit(): %v", err)
		return nil, http.StatusInternalServerError, ErrDatabaseError
	}

	return &msg, http.StatusCreated, nil
}

// GetMessages get messsages from a conversation. You can filter it by:
// `after` - get messages with ID bigger than `after`,
// `limit` - get maximum of `limit` messages,
// `orderBy` - get messages sorted by given order.
// This function will return messages, a boolean describes if there are more messages
// than the current result if you use `limit` option, or error if occurs.
// The status code is 200 if the operation is done successfully.
// The status code is 500 database error occurs.
func (d *DatabaseService) GetMessages(conversationId int64, after types.Optional[int64], limit types.Optional[int64], orderBy types.Optional[string]) ([]model.Message, bool, int, error) {
	if d.Status() != services.READY {
		d.logger.Errorfln("[%s] Database is not ready", d.Name())
		return nil, false, http.StatusInternalServerError, ErrDatabaseError
	}

	args := []any{}

	whereClauses := []string{
		"m.receiver_id = $1",
	}
	args = append(args, conversationId)

	argCount := 2
	if after.Valid {
		args = append(args, after.Value)
		whereClauses = append(whereClauses, fmt.Sprintf("m.id > $%d", argCount))
		argCount++
	}

	orderByPart := ""
	if orderBy.Valid {
		parts := strings.Split(orderBy.Value, ":")
		if len(parts) != 2 {
			d.logger.Debugfln("invalid orderBy value: %s", orderBy.Value)
			return nil, false, http.StatusInternalServerError, ErrDatabaseError
		}

		// orderByPart = "ORDER BY m.content ASC"
		// right now, key will always belong to message object
		key := fmt.Sprintf("m.%s", parts[0])
		order := parts[1]
		orderByPart = fmt.Sprintf("ORDER BY %s", fmt.Sprintf("%s %s", key, order))
	} else {
		orderByPart = "ORDER BY m.id DESC"
	}

	limitPart := ""
	if limit.Valid {
		// +1 so we can check if there are more messages
		args = append(args, limit.Value+1)
		limitPart = fmt.Sprintf("LIMIT $%d", argCount)
		argCount++
	}

	lastPart := strings.Join([]string{orderByPart, limitPart}, " ")
	lastPart = strings.TrimSpace(lastPart)

	query := fmt.Sprintf(`
		SELECT 
			m.id, m.content, m.type, m.created_at, m.updated_at, m.deleted_at, m.sender_id, receiver_id,
		    COALESCE(
            	json_agg(
                	json_build_object(
                    	'id', a.id,
						'thumb_url', a.thumb_url,
						'file_url', a.file_url,
						'deleted_at', a.deleted_at
                	)
            	) FILTER (WHERE a.id IS NOT NULL), '[]'
        	) AS attachments
    	FROM message.message m
    	LEFT JOIN message.attachment a ON m.id = a.message_id
    	WHERE %s
    	GROUP BY m.id
		%s;
	`, strings.Join(whereClauses, " AND "), lastPart)

	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	messageRows, err := d.client.Query(query, args...)
	if err != nil {
		d.logger.Errorfln("client.Exec(): %v", err)
		return nil, false, http.StatusInternalServerError, ErrDatabaseError
	}

	messages := []model.Message{}
	var rawAttachments json.RawMessage
	for messageRows.Next() {
		message := model.Message{}
		err = messageRows.Scan(&message.Id, &message.Content, &message.Type, &message.CreatedAt, &message.UpdatedAt, &message.DeletedAt, &message.SenderId, &message.ReceiverId, &rawAttachments)
		if err != nil {
			d.logger.Errorfln("messageRows.Scan(): %v", err)
			return nil, false, http.StatusInternalServerError, ErrDatabaseError
		}

		err = json.Unmarshal(rawAttachments, &message.Attachments)
		if err != nil {
			d.logger.Errorfln("json.Unmarshal(): %v", err)
			return nil, false, http.StatusInternalServerError, ErrDatabaseError
		}

		messages = append(messages, message)
	}

	hasMore := false
	if limit.Valid && len(messages) > int(limit.Value) {
		hasMore = true
		// remove the last one
		messages = messages[:len(messages)-1]
	}

	return messages, hasMore, http.StatusOK, nil
}

// GetAllMessages get all messages in the database. The function is currently used
// fro testing purposes. It will return messages or error if occurs.
func (d *DatabaseService) GetAllMessages() ([]model.Message, error) {
	if d.Status() != services.READY {
		d.logger.Errorfln("[%s] Database is not ready", d.Name())
		return nil, ErrDatabaseError
	}

	messageRows, err := d.client.Query(`
		SELECT
			m.id, m.content, m.type, m.created_at, m.updated_at, m.deleted_at, m.sender_id, m.receiver_id,
		    COALESCE(
            	json_agg(
                	json_build_object(
                    	'id', a.id,
						'thumb_url', a.thumb_url,
						'file_url', a.file_url,
						'deleted_at', a.deleted_at
                	)
            	) FILTER (WHERE a.id IS NOT NULL), '[]'
        	) AS attachments
    	FROM message.message m
    	LEFT JOIN message.attachment a ON m.id = a.message_id
    	GROUP BY m.id;
	`)

	if err != nil {
		return nil, err
	}

	messages := []model.Message{}
	var rawAttachments json.RawMessage
	for messageRows.Next() {
		message := model.Message{}
		err = messageRows.Scan(&message.Id, &message.Content, &message.Type, &message.CreatedAt, &message.UpdatedAt, &message.DeletedAt, &message.SenderId, &message.ReceiverId, &rawAttachments)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// CreateTemporaryData creates temporary data in the database for testing purposes.
func (d *DatabaseService) CreateTemporaryData(dataFile DataFile) error {
	tx, err := d.client.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if dataFile.UserFile != "" {
		userData, err := os.ReadFile(dataFile.UserFile)
		if err != nil {
			return err
		}
		var users []model.ChatUser
		err = json.Unmarshal(userData, &users)
		if err != nil {
			return err
		}
		fmt.Println("Creating temporary users")
		// Password is not hashed
		for _, user := range users {
			query := `INSERT INTO chat_user.chat_user (
            id, email, username, password, role, first_name, last_name, birthday, gender, phone_number, privacy, avatar, created_at, updated_at, deleted_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
        );`
			args := []any{user.Id, user.Email, user.Username, user.Password, user.Role, user.FirstName, user.LastName, user.Birthday, user.Gender, user.PhoneNumber, user.Privacy, user.Avatar, user.CreatedAt, user.UpdatedAt, user.DeletedAt}
			d.logger.Debugfln("query: %s; args: %v", helper.StripWS(query), args)
			_, err = tx.Exec(query, args...)
			if err != nil {
				return err
			}
		}
	}

	if dataFile.ConversationFile != "" {
		conversationData, err := os.ReadFile(dataFile.ConversationFile)
		if err != nil {
			return err
		}
		var conversations []model.Conversation
		err = json.Unmarshal(conversationData, &conversations)
		if err != nil {
			return err
		}
		fmt.Println("Creating temporary conversations")
		for _, conversation := range conversations {
			_, err = tx.Exec(`INSERT INTO conversation.conversation (
			id, type, created_at, deleted_at
		) VALUES (
			$1, $2, $3, $4
		);`, conversation.Id, conversation.Type, conversation.CreatedAt, conversation.DeletedAt)
			if err != nil {
				return err
			}
			for _, participant := range conversation.Participants {
				_, err = tx.Exec(`INSERT INTO conversation.participant (
					id, user_id, name, conversation_id, role
				) VALUES (
					$1, $2, $3, $4, $5
				);`, participant.Id, participant.UserId, participant.Name, conversation.Id, participant.Role)
				if err != nil {
					return err
				}
			}
			if conversation.Type == model.GROUP {
				if conversation.Group == nil {
					return fmt.Errorf("group conversation %d is missing group information", conversation.Id.Int64())
				}
				_, err = tx.Exec(`INSERT INTO conversation.group_chat (
			id, name, avatar, updated_at, conversation_id
		) VALUES (
			$1, $2, $3, $4, $5
		);`, conversation.Group.Id, conversation.Group.Name, conversation.Group.Avatar, conversation.Group.UpdatedAt, conversation.Id)
				if err != nil {
					return err
				}
			}
		}
	}

	if dataFile.ParticipantFile != "" {
		participantData, err := os.ReadFile(dataFile.ParticipantFile)
		if err != nil {
			return err
		}
		var participants []model.Participant
		err = json.Unmarshal(participantData, &participants)
		if err != nil {
			return err
		}

		fmt.Println("Creating temporary participants")
		for _, participant := range participants {
			_, err = tx.Exec(`INSERT INTO conversation.participant (
							id, user_id, name, conversation_id, role
						) VALUES (
							$1, $2, $3, $4, $5
						);`, participant.Id, participant.UserId, participant.Name, participant.ConversationId, participant.Role)
			if err != nil {
				return err
			}
		}
	}

	if dataFile.MessageFile != "" {
		messageData, err := os.ReadFile(dataFile.MessageFile)
		if err != nil {
			return err
		}
		var messages []model.Message
		err = json.Unmarshal(messageData, &messages)
		if err != nil {
			return err
		}

		fmt.Println("Creating temporary messages")
		for _, message := range messages {
			_, err = tx.Exec(`INSERT INTO message.message (
				id, content, sender_id, receiver_id, created_at, updated_at, deleted_at, type
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8
			);`, message.Id, message.Content, message.SenderId, message.ReceiverId, message.CreatedAt, message.UpdatedAt, message.DeletedAt, message.Type)
			if err != nil {
				return err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil

}

// ClearAllData clears all data in the database. This function is used for
// testing purposes.
//
// FIXED (2025-07-20):
// Previously, this function only cleared the `chat_user` tables. It assumed
// that the `conversation` tables were deleted when users were deleted. But
// now, conversations are not dependent on users, so we need to clear the
// `conversation` tables as well.
func (d *DatabaseService) ClearAllData() error {
	queries := []string{
		`DELETE FROM chat_user.chat_user;`,
		`DELETE FROM conversation.conversation;`,
	}

	for _, query := range queries {
		d.logger.Debugfln("query: %s", query)
		_, err := d.client.Exec(query)
		if err != nil {
			return err
		}
	}

	return nil
}
