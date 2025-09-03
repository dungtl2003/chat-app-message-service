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
	"os"
	"strings"

	_ "github.com/lib/pq" // postgresql driver support
)

type DataFile struct {
	UserFile         string
	ConversationFile string
	MessageFile      string
	AssetFile        string
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

// New creates a new database connection. The function returns a database
// connection and an error. Remember to call Close() when done to release resources.
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

// CreateMessage create a new message. It will return the created message and
// error if occurs.
// ErrDatabaseError will be returned if database error occurs.
func (d *DatabaseService) CreateMessage(message model.Message) (*model.Message, error) {
	if d.Status() != services.READY {
		d.logger.Errorfln("[%s] Database is not ready", d.Name())
		return nil, ErrDatabaseError
	}

	tx, err := d.client.Begin()
	if err != nil {
		d.logger.Errorfln("client.Begin(): %v", err)
		return nil, ErrDatabaseError
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `
		INSERT INTO message.message (
			id, content, type, created_at, updated_at, deleted_at, sender_id, receiver_id, reply_to_message_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		);`
	args := []any{message.Id, message.Content, message.Type, message.CreatedAt, message.UpdatedAt, message.DeletedAt, message.SenderId, message.ReceiverId, message.ReplyToMessageId}
	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	_, err = tx.Exec(query, args...)
	if err != nil {
		d.logger.Errorfln("tx.Exec(): %v", err)
		return nil, ErrDatabaseError
	}

	args = make([]any, len(message.Attachments)*6)
	argsCount := 1
	values := make([]string, len(message.Attachments))
	for i, attachment := range message.Attachments {
		values[i] = fmt.Sprintf(`($%d, $%d, $%d, $%d, $%d, $%d)`, argsCount, argsCount+1, argsCount+2, argsCount+3, argsCount+4, argsCount+5)
		args[argsCount-1] = attachment.Id
		args[argsCount] = attachment.AssetId
		args[argsCount+1] = attachment.DeletedAt
		args[argsCount+2] = attachment.MessageId
		args[argsCount+3] = attachment.Position
		args[argsCount+4] = attachment.Type
		argsCount += 6
	}
	query = fmt.Sprintf(`
		INSERT INTO message.attachment (
			id, asset_id, deleted_at, message_id, position, type
		) VALUES %s;
	`, strings.Join(values, ", "))
	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	_, err = tx.Exec(query, args...)
	if err != nil {
		d.logger.Errorfln("tx.Exec(): %v", err)
		return nil, ErrDatabaseError
	}

	query = `
		SELECT 
			m.id, m.content, m.type, m.created_at, m.updated_at, m.deleted_at, m.sender_id, m.receiver_id, m.reply_to_message_id,
		    COALESCE(
            	json_agg(
                	json_build_object(
                    	'id', a.id,
						'asset_id', a.asset_id,
						'deleted_at', a.deleted_at,
						'message_id', a.message_id,
						'position', a.position,
						'type', a.type
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
	err = messageRow.Scan(
		&msg.Id, &msg.Content, &msg.Type, &msg.CreatedAt, &msg.UpdatedAt, &msg.DeletedAt, &msg.SenderId, &msg.ReceiverId, &msg.ReplyToMessageId,
		&rawAttachments)
	if err != nil {
		d.logger.Errorfln("messageRow.Scan(): %v", err)
		return nil, ErrDatabaseError
	}

	err = json.Unmarshal(rawAttachments, &msg.Attachments)
	if err != nil {
		d.logger.Errorfln("json.Unmarshal(): %v", err)
		return nil, ErrDatabaseError
	}

	err = tx.Commit()
	if err != nil {
		d.logger.Errorfln("tx.Commit(): %v", err)
		return nil, ErrDatabaseError
	}

	return &msg, nil
}

// GetMessages get messsages from a conversation. You can filter it by:
// `after` - get messages with ID bigger than `after`,
// `limit` - get maximum of `limit` messages,
// This function will return the end cursor, list of messages, hasMore flag
// and error if occurs.
// ErrDatabaseError will be returned if database error occurs.
func (d *DatabaseService) GetMessages(conversationId int64, after types.Optional[int64], limit types.Optional[int64]) (int64, []model.Message, bool, error) {
	if d.Status() != services.READY {
		d.logger.Errorfln("[%s] Database is not ready", d.Name())
		return 0, nil, false, ErrDatabaseError
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

	limitPart := ""
	if limit.Valid {
		// +1 so we can check if there are more messages
		args = append(args, limit.Value+1)
		limitPart = fmt.Sprintf("LIMIT $%d", argCount)
		argCount++
	}

	query := fmt.Sprintf(`
		SELECT 
			m.id, m.content, m.type, m.created_at, m.updated_at, m.deleted_at, m.sender_id, receiver_id, reply_to_message_id,
		    COALESCE(
            	json_agg(
                	json_build_object(
                    	'id', a.id,
						'asset_id', a.asset_id,
						'deleted_at', a.deleted_at,
						'message_id', a.message_id,
						'position', a.position,
						'type', a.type
                	)
            	) FILTER (WHERE a.id IS NOT NULL), '[]'
        	) AS attachments
    	FROM message.message m
    	LEFT JOIN message.attachment a ON m.id = a.message_id
    	WHERE %s
    	GROUP BY m.id
		ORDER BY m.id DESC
		%s;
	`, strings.Join(whereClauses, " AND "), limitPart)

	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	messageRows, err := d.client.Query(query, args...)
	if err != nil {
		d.logger.Errorfln("client.Exec(): %v", err)
		return 0, nil, false, ErrDatabaseError
	}

	messages := []model.Message{}
	prevCursor := types.NewJsonInt64(0)
	endCursor := types.NewJsonInt64(0)
	var rawAttachments json.RawMessage
	for messageRows.Next() {
		prevCursor = endCursor
		message := model.Message{}
		err = messageRows.Scan(
			&message.Id, &message.Content, &message.Type, &message.CreatedAt, &message.UpdatedAt, &message.DeletedAt, &message.SenderId, &message.ReceiverId, &message.ReplyToMessageId,
			&rawAttachments,
		)
		if err != nil {
			d.logger.Errorfln("messageRows.Scan(): %v", err)
			return 0, nil, false, ErrDatabaseError
		}

		err = json.Unmarshal(rawAttachments, &message.Attachments)
		if err != nil {
			d.logger.Errorfln("json.Unmarshal(): %v", err)
			return 0, nil, false, ErrDatabaseError
		}

		messages = append(messages, message)
		endCursor = message.Id
	}

	hasMore := false
	if limit.Valid && len(messages) > int(limit.Value) {
		hasMore = true
		// remove the last one
		messages = messages[:len(messages)-1]
		endCursor = prevCursor
	}

	return endCursor.Int64(), messages, hasMore, nil
}

// GetAllMessages get all messages in the database. The function is currently used
// for testing purposes. It will return messages or error if occurs.
func (d *DatabaseService) GetAllMessages() ([]model.Message, error) {
	if d.Status() != services.READY {
		d.logger.Errorfln("[%s] Database is not ready", d.Name())
		return nil, ErrDatabaseError
	}

	messageRows, err := d.client.Query(`
		SELECT
			m.id, m.content, m.type, m.created_at, m.updated_at, m.deleted_at, m.sender_id, m.receiver_id, m.reply_to_message_id,
		    COALESCE(
            	json_agg(
                	json_build_object(
                    	'id', a.id,
						'asset_id', a.asset_id,
						'deleted_at', a.deleted_at,
						'message_id', a.message_id,
						'position', a.position,
						'type', a.type
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
		err = messageRows.Scan(
			&message.Id, &message.Content, &message.Type, &message.CreatedAt, &message.UpdatedAt, &message.DeletedAt, &message.SenderId, &message.ReceiverId, &message.ReplyToMessageId,
			&rawAttachments)
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

	// This must be done first to ensure that the assets are created before so
	// that other data can reference them.
	if dataFile.AssetFile != "" {
		assetData, err := os.ReadFile(dataFile.AssetFile)
		if err != nil {
			return err
		}
		var assets []model.Asset
		err = json.Unmarshal(assetData, &assets)
		if err != nil {
			return err
		}
		d.logger.Debugfln("[%s] Loaded %d assets from file: %s", d.Name(), len(assets), dataFile.AssetFile)
		for _, asset := range assets {
			query := `INSERT INTO media.asset (
				id, public_id, width, height, format, resource_type, created_at, bytes, url, secure_url, asset_folder, original_filename, api_key
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
			);`
			args := []any{
				asset.Id, asset.PublicId, asset.Width, asset.Height, asset.Format, asset.ResourceType, asset.CreatedAt, asset.Bytes, asset.Url, asset.SecureUrl, asset.AssetFolder, asset.OriginalFilename, asset.ApiKey,
			}
			d.logger.Debugfln("[%s] Executing query: %s with args: %v", d.Name(), helper.StripWS(query), args)
			_, err = tx.Exec(query, args...)
			if err != nil {
				return err
			}
		}
	}

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

		d.logger.Debugfln("[%s] Loaded %d users from file: %s", d.Name(), len(users), dataFile.UserFile)
		for _, user := range users {
			query := `INSERT INTO chat_user.chat_user (
            id, email, username, password, role, first_name, last_name, birthday, gender, phone_number, privacy, avatar_id, created_at, updated_at, deleted_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
        );`
			args := []any{user.Id, user.Email, user.Username, user.Password, user.Role, user.FirstName, user.LastName, user.Birthday, user.Gender, user.PhoneNumber, user.Privacy, user.AvatarId, user.CreatedAt, user.UpdatedAt, user.DeletedAt}
			d.logger.Debugfln("[%s] Executing query: %s with args: %v", d.Name(), helper.StripWS(query), args)
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

		d.logger.Debugfln("[%s] Loaded %d conversations from file: %s", d.Name(), len(conversations), dataFile.ConversationFile)
		for _, conversation := range conversations {
			query := `INSERT INTO conversation.conversation (
				id, type, created_at, deleted_at
			) VALUES (
				$1, $2, $3, $4
			);`
			args := []any{conversation.Id, conversation.Type, conversation.CreatedAt, conversation.DeletedAt}
			d.logger.Debugfln("[%s] Executing query: %s with args: %v", d.Name(), helper.StripWS(query), args)
			_, err = tx.Exec(query, args...)
			if err != nil {
				return err
			}
			for _, participant := range conversation.Participants {
				query = `INSERT INTO conversation.participant (
					id, user_id, nickname, conversation_id, role, avatar_id, username, email, first_name, last_name
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
				);`
				args = []any{participant.Id, participant.UserId, participant.Nickname, conversation.Id, participant.Role, participant.AvatarId, participant.Username, participant.Email, participant.FirstName, participant.LastName}
				d.logger.Debugfln("[%s] Executing query: %s with args: %v", d.Name(), helper.StripWS(query), args)
				_, err = tx.Exec(query, args...)
				if err != nil {
					return err
				}
			}
			if conversation.Type == model.GROUP {
				if conversation.Group == nil {
					return fmt.Errorf("group conversation %d is missing group information", conversation.Id.Int64())
				}
				query = `INSERT INTO conversation.group_chat (
					id, name, avatar_id, conversation_id
				) VALUES (
					$1, $2, $3, $4
				);`
				args = []any{conversation.Group.Id, conversation.Group.Name, conversation.Group.AvatarId, conversation.Id}
				d.logger.Debugfln("[%s] Executing query: %s with args: %v", d.Name(), helper.StripWS(query), args)
				_, err = tx.Exec(query, args...)
				if err != nil {
					return err
				}
			}
		}
	}

	if dataFile.MessageFile != "" {
		messageData, err := os.ReadFile(dataFile.MessageFile)
		if err != nil {
			return err
		}
		messages := []model.Message{}
		err = json.Unmarshal(messageData, &messages)
		if err != nil {
			return err
		}

		d.logger.Debugfln("[%s] Loaded %d messages from file: %s", d.Name(), len(messages), dataFile.MessageFile)
		for _, message := range messages {
			query := `INSERT INTO message.message (
				id, content, sender_id, receiver_id, created_at, updated_at, deleted_at, type
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8
			);`
			args := []any{message.Id, message.Content, message.SenderId, message.ReceiverId, message.CreatedAt, message.UpdatedAt, message.DeletedAt, message.Type}
			d.logger.Debugfln("[%s] Executing query: %s with args: %v", d.Name(), helper.StripWS(query), args)
			_, err = tx.Exec(query, args...)
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
