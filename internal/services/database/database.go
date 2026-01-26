package database

import (
	"context"
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

	"github.com/lib/pq"
	_ "github.com/lib/pq" // postgresql driver support
)

type DataFile struct {
	UserFile         string
	ConversationFile string
	MessageFile      string
	AssetFile        string
}

var (
	ErrDatabaseError      = fmt.Errorf("database error")
	ErrDatabaseNotRunning = fmt.Errorf("database is not running")
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
	if d.status != services.ServiceStopped {
		// check if the database connection is still alive
		if err := d.client.Ping(); err != nil {
			d.logger.Errorfln("[%s] Database connection is not alive: %v", d.Name(), err)
			d.status = services.ServiceError
		} else {
			d.status = services.ServiceReady
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
		status: services.ServiceReady,
	}

	d.logger.Infofln("[%s] Database connection created", d.Name())
	d.logger.Infofln("[%s] Running", d.Name())
	return d, nil
}

// Close closes the database connection. The function returns an error.
func (d *DatabaseService) Close() error {
	if d.Status() == services.ServiceStopped {
		d.logger.Errorfln("[%s] Database connection is already closed", d.Name())
		return nil
	}

	err := d.client.Close()
	if err != nil {
		d.logger.Errorfln("[%s] Failed to close database connection: %v", d.Name(), err)
		d.status = services.ServiceError
	} else {
		d.logger.Infofln("[%s] Database connection closed", d.Name())
		d.status = services.ServiceStopped
	}

	return err
}

// FetchPendingOutboxEvents fetches pending outbox events for processing.
// `ErrDatabaseNotRunning` will be returned if the database is not running.
func (d *DatabaseService) FetchPendingOutboxEvents(
	ctx context.Context,
	limit int64,
) ([]model.MessageOutbox, error) {
	if d.Status() == services.ServiceStopped {
		return nil, ErrDatabaseNotRunning
	}
	if d.Status() != services.ServiceReady {
		d.logger.Warnfln("[%s] Database is not ready", d.Name())
	}

	// 1. SELECT pending events
	// 2. ORDER BY id ASC (FIFO)
	// 3. DISTINCT ON (conversation_id) -> Ensures we only pick the very first message for a convo.
	//    If the first message is stuck/locked, we skip the WHOLE conversation.
	// 4. FOR UPDATE SKIP LOCKED -> Allows other instances to pick different conversations.
	query := `
    SELECT 
		id, 
		message_id, 
		conversation_id, 
		conversation_event_id, 
		payload,
		status, 
		created_at, 
		processed_at, 
		retry_count, 
		last_error, 
		next_retry_at
    FROM message.message_outbox
    WHERE id IN (
        SELECT DISTINCT ON (conversation_id) id
        FROM message.message_outbox
        WHERE status = $1 AND next_retry_at <= NOW()
        ORDER BY conversation_id, id ASC
    )
    LIMIT $2
    FOR UPDATE SKIP LOCKED
    `
	args := []any{model.OUTBOX_PENDING, limit}

	// d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	rows, err := d.client.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.MessageOutbox
	for rows.Next() {
		var e model.MessageOutbox
		// Scan based on your actual columns
		if err := rows.Scan(
			&e.Id,
			&e.MessageId,
			&e.ConversationId,
			&e.ConversationEventId,
			&e.Payload,
			&e.Status,
			&e.CreatedAt,
			&e.ProcessedAt,
			&e.RetryCount,
			&e.LastError,
			&e.NextRetryAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// MarkOutboxEventSent marks an outbox event as processed.
// `ErrDatabaseNotRunning` will be returned if the database is not running.
func (d *DatabaseService) MarkOutboxEventSent(
	ctx context.Context,
	id int64,
) error {
	if d.Status() == services.ServiceStopped {
		return ErrDatabaseNotRunning
	}
	if d.Status() != services.ServiceReady {
		d.logger.Warnfln("[%s] Database is not ready", d.Name())
	}

	query := `UPDATE message.message_outbox SET status = $1, processed_at = NOW() WHERE id = $2`
	args := []any{model.OUTBOX_SENT, id}
	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	_, err := d.client.ExecContext(ctx, query, args...)
	return err
}

// MarkOutboxEventFailed marks an outbox event as failed, increments the retry count,
// and schedules the next retry time using exponential backoff.
// `ErrDatabaseNotRunning` will be returned if the database is not running.
func (d *DatabaseService) MarkOutboxEventFailed(ctx context.Context, id int64, errRaw error) error {
	if d.Status() == services.ServiceStopped {
		return ErrDatabaseNotRunning
	}
	if d.Status() != services.ServiceReady {
		d.logger.Warnfln("[%s] Database is not ready", d.Name())
	}

	// Exponential backoff or fixed delay (e.g., 5 seconds)
	// We increment retry_count and push next_retry_at into the future
	query := `
        UPDATE message.message_outbox 
        SET retry_count = retry_count + 1,
            last_error = $1,
            next_retry_at = NOW() + (INTERVAL '1 second' * power(2, retry_count))
        WHERE id = $2`
	args := []any{errRaw.Error(), id}

	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	_, err := d.client.ExecContext(ctx, query, args...)
	return err
}

// CreateMessage create a new message. It will return the created message and
// error if occurs.
// `ErrDatabaseNotRunning` will be returned if the database is not running.
func (d *DatabaseService) CreateMessage(
	ctx context.Context,
	message model.Message,
	idempotencyKey string,
	conversationEventId int64,
) (*model.Message, error) {
	if d.Status() == services.ServiceStopped {
		return nil, ErrDatabaseNotRunning
	}
	if d.Status() != services.ServiceReady {
		d.logger.Warnfln("[%s] Database is not ready", d.Name())
	}

	tx, err := d.client.Begin()
	if err != nil {
		d.logger.Errorfln("client.Begin(): %v", err)
		return nil, err
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
	args := []any{
		message.Id,
		message.Content,
		message.Type,
		message.CreatedAt,
		message.UpdatedAt,
		message.DeletedAt,
		message.SenderId,
		message.ReceiverId,
		message.ReplyToMessageId,
	}
	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	if len(message.Attachments) > 0 {
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
		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
	}

	query = `
		SELECT 
			m.id, m.content, m.type, m.created_at, m.updated_at, m.deleted_at, m.sender_id, m.receiver_id, m.reply_to_message_id, m.version,
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
	messageRow := tx.QueryRowContext(ctx, query, args...)
	err = messageRow.Scan(
		&msg.Id,
		&msg.Content,
		&msg.Type,
		&msg.CreatedAt,
		&msg.UpdatedAt,
		&msg.DeletedAt,
		&msg.SenderId,
		&msg.ReceiverId,
		&msg.ReplyToMessageId,
		&msg.Version,
		&rawAttachments,
	)
	if err != nil {
		return nil, err
	}
	msg.IdempotencyKey = idempotencyKey

	err = json.Unmarshal(rawAttachments, &msg.Attachments)
	if err != nil {
		return nil, err
	}

	// Attach asset objects from the input message parameter to the resulting message
	assetMap := make(map[int64]*model.Asset)
	for _, att := range message.Attachments {
		if att.Asset != nil {
			assetMap[att.AssetId.Int64()] = att.Asset
		}
	}
	for i := range msg.Attachments {
		if asset, ok := assetMap[msg.Attachments[i].AssetId.Int64()]; ok {
			msg.Attachments[i].Asset = asset
		}
	}

	payloadData := model.MessageOutboxPayload{
		Message:        msg,
		IdempotencyKey: idempotencyKey,
	}
	payloadBytes, err := json.Marshal(payloadData)
	if err != nil {
		return nil, err
	}
	payloadJson := types.NewJson(payloadBytes)

	query = `
		INSERT INTO message.message_outbox (
			id, message_id, conversation_id, conversation_event_id, payload, status, created_at, next_retry_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		);`
	args = []any{
		msg.Id,
		msg.Id,
		msg.ReceiverId,
		conversationEventId,
		payloadJson,
		model.OUTBOX_PENDING,
		msg.CreatedAt,
		msg.CreatedAt, // first retry at the time of message creation
	}
	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
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
	if d.Status() == services.ServiceStopped {
		d.logger.Errorfln("[%s] Database is not running", d.Name())
		return 0, nil, false, ErrDatabaseError
	}
	if d.Status() != services.ServiceReady {
		d.logger.Warnfln("[%s] Database is not ready", d.Name())
	}

	args := []any{}

	whereClauses := []string{
		"m.receiver_id = $1",
	}
	args = append(args, conversationId)

	argCount := 2
	if after.Valid {
		args = append(args, after.Value)
		// get messages with ID less than `after` (descending order)
		whereClauses = append(whereClauses, fmt.Sprintf("m.id < $%d", argCount))
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

// GetMessageById get a message by its ID. It will return the message and error
// if occurs. If the message is not found, it will return nil, nil.
// `ErrDatabaseError` will be returned if database error occurs.
func (d *DatabaseService) GetMessageById(messageId int64) (*model.Message, error) {
	if d.Status() == services.ServiceStopped {
		d.logger.Errorfln("[%s] Database is not running", d.Name())
		return nil, ErrDatabaseError
	}
	if d.Status() != services.ServiceReady {
		d.logger.Warnfln("[%s] Database is not ready", d.Name())
	}

	query := `
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
	args := []any{messageId}
	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	var msg model.Message
	var rawAttachments json.RawMessage
	messageRow := d.client.QueryRow(query, args...)
	err := messageRow.Scan(
		&msg.Id, &msg.Content, &msg.Type, &msg.CreatedAt, &msg.UpdatedAt, &msg.DeletedAt, &msg.SenderId, &msg.ReceiverId, &msg.ReplyToMessageId,
		&rawAttachments)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		d.logger.Errorfln("messageRow.Scan(): %v", err)
		return nil, ErrDatabaseError
	}

	err = json.Unmarshal(rawAttachments, &msg.Attachments)
	if err != nil {
		d.logger.Errorfln("json.Unmarshal(): %v", err)
		return nil, ErrDatabaseError
	}

	return &msg, nil
}

// GetOutboxEventById get an outbox event by its ID. It will return the outbox event and error
// if occurs. If the outbox event is not found, it will return nil, nil.
func (d *DatabaseService) GetOutboxEventById(ctx context.Context, id int64) (*model.MessageOutbox, error) {
	if d.Status() == services.ServiceStopped {
		return nil, ErrDatabaseNotRunning
	}
	if d.Status() != services.ServiceReady {
		d.logger.Warnfln("[%s] Database is not ready", d.Name())
	}

	query := `
		SELECT
			id, message_id, conversation_id, conversation_event_id, payload, status, created_at, processed_at, retry_count, last_error, next_retry_at
		FROM message.message_outbox
		WHERE id = $1;
	`
	args := []any{id}
	d.logger.Debugfln("SQL command: %s, arguments: %#v", helper.StripWS(query), args)
	var e model.MessageOutbox
	err := d.client.QueryRowContext(ctx, query, args...).Scan(
		&e.Id,
		&e.MessageId,
		&e.ConversationId,
		&e.ConversationEventId,
		&e.Payload,
		&e.Status,
		&e.CreatedAt,
		&e.ProcessedAt,
		&e.RetryCount,
		&e.LastError,
		&e.NextRetryAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

// GetUsersByIds retrieves users from the database by user IDs.
// The function returns a list of users (with deleted_at =IS NULL) and an error if occurs.
// `ErrDatabaseError` will be returned if there is an error while executing the query.
func (d *DatabaseService) GetUsersByIds(
	ctx context.Context,
	userIds []int64,
) ([]model.ChatUser, error) {
	if d.Status() == services.ServiceStopped {
		d.logger.Errorfln("[%s] Database is not running", d.Name())
		return nil, ErrDatabaseError
	}
	if d.Status() != services.ServiceReady {
		d.logger.Warnfln("[%s] Database is not ready", d.Name())
	}
	if len(userIds) == 0 {
		return []model.ChatUser{}, nil
	}

	// we only get basic user info for privacy reason
	query := `SELECT 
		u.id, 
		u.username, 
		u.role, 
		u.first_name, 
		u.last_name, 
		u.full_name, 
		u.birthday, 
		u.gender, 
		u.avatar_id, 
		u.created_at, 
		u.updated_at, 
		u.version
	FROM chat_user.chat_user u
	WHERE u.id = ANY($1) AND u.deleted_at IS NULL;`
	args := []any{pq.Array(userIds)}
	d.logger.Debugfln("query: %s --- args: %v", helper.StripWS(query), args)
	rows, err := d.client.QueryContext(ctx, query, args...)
	if err != nil {
		d.logger.Errorfln("client.QueryContext(): %v", err)
		return nil, ErrDatabaseError
	}
	defer rows.Close()

	var users []model.ChatUser
	for rows.Next() {
		user := model.ChatUser{}
		err := rows.Scan(
			&user.Id,
			&user.Username,
			&user.Role,
			&user.FirstName,
			&user.LastName,
			&user.FullName,
			&user.Birthday,
			&user.Gender,
			&user.AvatarId,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.Version,
		)
		if err != nil {
			d.logger.Errorfln("rows.Scan(): %v", err)
			return nil, ErrDatabaseError
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		d.logger.Errorfln("rows.Err(): %v", err)
		return nil, ErrDatabaseError
	}

	return users, nil
}

// GetParticipantsByConversationIdAndUserIds retrieves participants from the database by conversation ID and user IDs.
func (d *DatabaseService) GetParticipantsByConversationIdAndUserIds(
	ctx context.Context,
	conversationId int64,
	userIds []int64,
) ([]model.Participant, error) {
	if d.Status() == services.ServiceStopped {
		d.logger.Errorfln("[%s] Database is not running", d.Name())
		return nil, ErrDatabaseError
	}
	if d.Status() != services.ServiceReady {
		d.logger.Warnfln("[%s] Database is not ready", d.Name())
	}

	query := `SELECT
		id,
		user_id,
		nickname,
		role,
		conversation_id
	FROM conversation.participant
	WHERE conversation_id = $1 AND user_id = ANY($2);`
	args := []any{conversationId, pq.Array(userIds)}

	d.logger.Debugfln("query: %s --- args: %v", helper.StripWS(query), args)
	rows, err := d.client.QueryContext(ctx, query, args...)
	if err != nil {
		d.logger.Errorfln("client.QueryContext(): %v", err)
		return nil, ErrDatabaseError
	}
	defer rows.Close()

	var participants []model.Participant
	for rows.Next() {
		participant := model.Participant{}
		err := rows.Scan(
			&participant.Id,
			&participant.UserId,
			&participant.Nickname,
			&participant.Role,
			&participant.ConversationId,
		)
		if err != nil {
			d.logger.Errorfln("rows.Scan(): %v", err)
			return nil, ErrDatabaseError
		}

		participants = append(participants, participant)
	}

	if err = rows.Err(); err != nil {
		d.logger.Errorfln("rows.Err(): %v", err)
		return nil, ErrDatabaseError
	}

	return participants, nil
}

// GetAllMessages get all messages in the database. The function is currently used
// for testing purposes. It will return messages or error if occurs.
func (d *DatabaseService) GetAllMessages() ([]model.Message, error) {
	if d.Status() == services.ServiceStopped {
		d.logger.Errorfln("[%s] Database is not running", d.Name())
		return nil, ErrDatabaseError
	}
	if d.Status() != services.ServiceReady {
		d.logger.Warnfln("[%s] Database is not ready", d.Name())
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
					id, user_id, nickname, conversation_id, role
				) VALUES (
					$1, $2, $3, $4, $5
				);`
				args = []any{participant.Id, participant.UserId, participant.Nickname, conversation.Id, participant.Role}
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
					name, avatar_id, conversation_id
				) VALUES (
					$1, $2, $3
				);`
				args = []any{conversation.Group.Name, conversation.Group.AvatarId, conversation.Id}
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
		`DELETE FROM media.asset;`,
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
