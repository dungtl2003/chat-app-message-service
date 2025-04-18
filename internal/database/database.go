package database

import (
	"database/sql"
	"dungtl2003/chat-app-message-service/internal/logging"
	"fmt"

	_ "github.com/lib/pq" // postgresql driver support
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
