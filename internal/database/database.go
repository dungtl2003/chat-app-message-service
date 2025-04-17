package database

import (
	"database/sql"
	"dungtl2003/chat-app-message-service/internal/logging"

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
