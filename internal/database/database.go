package database

import (
	"database/sql"
	"dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/types"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	_ "github.com/lib/pq" // postgresql driver support
)

var (
	ErrDatabaseError = fmt.Errorf("database error")
)

type Database struct {
	client *sql.DB
	logger *logging.LoggerWrapper
}

// New creates a new database connection. The function returns a database connection and an error.
func New(url string, logger *logging.LoggerWrapper) (*Database, error) {
	client, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	return &Database{
		client: client,
		logger: logger,
	}, nil
}

// Close closes the database connection. The function returns an error.
func (d *Database) Close() error {
	d.logger.Info("closing database connection")
	if err := d.client.Close(); err != nil {
		d.logger.Errorfln("error when closing database connection: %v", err)
		return err
	} else {
		d.logger.Info("database connection closed")
		return nil
	}
}

// CreateMessage create a new message. It will return the created message, status
// and error if occurs.
// The status code is 200 if the operation is done successfully.
// The status code is 500 database error occurs.
func (d *Database) CreateMessage(message model.Message) (*model.Message, int, error) {
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
func (d *Database) GetMessages(conversationId int64, after types.Optional[int64], limit types.Optional[int64], orderBy types.Optional[string]) ([]model.Message, bool, int, error) {
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
func (d *Database) GetAllMessages() ([]model.Message, error) {
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

// Snapshot creates a snapshot of the current database state. The function is currently used for testing purposes.
// The function returns an error. You can use Rollback() to revert the database to the state before the snapshot.
func (d *Database) Snapshot() error {
	tx, err := d.client.Begin()
	if err != nil {
		d.logger.Error("error when starting transaction", "error", err)
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	cmds := []string{
		`CREATE TABLE IF NOT EXISTS chat_user.chat_user_snapshot AS SELECT * FROM chat_user.chat_user WHERE false;`,             // create an empty table
		`CREATE TABLE IF NOT EXISTS message.message_snapshot AS SELECT * FROM message.message WHERE false;`,                     // create an empty table
		`CREATE TABLE IF NOT EXISTS conversation.conversation_snapshot AS SELECT * FROM conversation.conversation WHERE false;`, // create an empty table
		`CREATE TABLE IF NOT EXISTS conversation.participant_snapshot AS SELECT * FROM conversation.participant WHERE false;`,   // create an empty table

		`DELETE FROM chat_user.chat_user_snapshot;`,
		`DELETE FROM message.message_snapshot;`,
		`DELETE FROM conversation.conversation_snapshot;`,
		`DELETE FROM conversation.participant_snapshot;`,

		`INSERT INTO chat_user.chat_user_snapshot SELECT * FROM chat_user.chat_user;`,
		`INSERT INTO message.message_snapshot SELECT * FROM message.message;`,
		`INSERT INTO conversation.conversation_snapshot SELECT * FROM conversation.conversation;`,
		`INSERT INTO conversation.participant_snapshot SELECT * FROM conversation.participant;`,
	}

	for _, cmd := range cmds {
		_, err = d.client.Exec(cmd)
		if err != nil {
			msg := fmt.Sprintf("error when trying to create snapshot: error when executing command: %s: %v", cmd, err)
			d.logger.Error(msg)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		msg := fmt.Sprintf("error when trying to create snapshot: error when committing transaction: %v", err)
		d.logger.Error(msg)
		return err
	}

	return nil
}

// Rollback rolls back the database to the state before the snapshot. The function is currently used for testing purposes.
// The function returns an error. You must call Snapshot() before calling this function.
func (d *Database) Rollback() error {
	tx, err := d.client.Begin()
	if err != nil {
		d.logger.Error("error when starting transaction", "error", err)
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	cmds := []string{
		`DELETE FROM chat_user.chat_user;`,

		`INSERT INTO chat_user.chat_user SELECT * FROM chat_user.chat_user_snapshot;`,
		`INSERT INTO conversation.conversation SELECT * FROM conversation.conversation_snapshot;`,
		`INSERT INTO conversation.participant SELECT * FROM conversation.participant_snapshot;`,
		`INSERT INTO message.message SELECT * FROM message.message_snapshot;`,

		`DROP TABLE chat_user.chat_user_snapshot;`,
		`DROP TABLE message.message_snapshot;`,
		`DROP TABLE conversation.conversation_snapshot;`,
		`DROP TABLE conversation.participant_snapshot;`,
	}

	for _, cmd := range cmds {
		_, err = d.client.Exec(cmd)
		if err != nil {
			msg := fmt.Sprintf("error when rolling back: error when executing command: %s: %v", cmd, err)
			d.logger.Error(msg)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		msg := fmt.Sprintf("error when rolling back: error when committing transaction: %v", err)
		d.logger.Error(msg)
		return err
	}

	return nil
}
