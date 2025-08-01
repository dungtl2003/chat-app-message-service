package tests

import (
	"dungtl2003/chat-app-message-service/internal/config"
	"dungtl2003/chat-app-message-service/internal/httpclient"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type SnowflakeConfig struct {
	Addr    string
	CertDir string
}

type Helper struct {
	Db                *database.DatabaseService
	Client            *httpclient.HttpClient
	Logger            *logging.LoggerWrapper
	SnowflakeConfig   *SnowflakeConfig
	DataFileDir       string
	MessageServiceURL string
}

const (
	JWT_USER_ID_2 = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXAiOiJKV1QiLCJzdWIiOiJ1c2VyOjIiLCJhdWQiOlsiaW50ZXJuYWwtc2VydmljZSJdLCJpYXQiOjE3MDY4NzA0MDAsImV4cCI6MTcwNjg3NDAwMH0.xQz5Ytlyuibsdcuh3LB-uNeFnK_DCkjKdw7eG0gy51g"
)

func NewHelper() *Helper {
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config.LoadConfig(): %v", err)
	}

	dbURL, has := os.LookupEnv("ADMIN_DATABASE_URL")
	if !has {
		log.Fatalf("Error when getting ADMIN_DATABASE_URL")
	}

	dataFileDir, bool := os.LookupEnv("DATA_FILE_DIR")
	if !bool {
		log.Fatal("DATA_FILE_DIR is not set")
	}

	logger, err := logging.NewLogger(config.LogConfig.Level, config.LogConfig.Kind)
	if err != nil {
		log.Fatalf("logging.NewLogger(): %v", err)
	}
	loggerWrapper := logging.NewLoggerWrapper(logger)
	if err != nil {
		log.Fatalf("logging.NewLoggerWrapper(): %v", err)
	}

	msgServiceURL, has := os.LookupEnv("MESSAGE_SERVICE_URL")
	if !has {
		log.Fatalf("Error when getting MESSAGE_SERVICE_URL")
	}

	db, err := database.New(dbURL, loggerWrapper)
	if err != nil {
		log.Fatalf("Error when creating admin database: %v", err)
	}

	client := httpclient.NewWithConfig(&http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: 55 * time.Second,
		},
		Timeout: 55 * time.Second,
	})

	helper := &Helper{
		Db:     db,
		Logger: loggerWrapper,
		Client: client,
		SnowflakeConfig: &SnowflakeConfig{
			Addr:    config.SnowflakeConfig.Addr,
			CertDir: config.SnowflakeConfig.CertDir,
		},
		MessageServiceURL: msgServiceURL,
		DataFileDir:       dataFileDir,
	}

	loggerWrapper.Info("Helper initialized")
	return helper
}

func (h *Helper) CreateTemporaryData(dataFile database.DataFile) error {
	if dataFile.UserFile != "" {
		dataFile.UserFile = fmt.Sprintf("%s/%s", h.DataFileDir, dataFile.UserFile)
	}
	if dataFile.ConversationFile != "" {
		dataFile.ConversationFile = fmt.Sprintf("%s/%s", h.DataFileDir, dataFile.ConversationFile)
	}
	if dataFile.ParticipantFile != "" {
		dataFile.ParticipantFile = fmt.Sprintf("%s/%s", h.DataFileDir, dataFile.ParticipantFile)
	}
	if dataFile.MessageFile != "" {
		dataFile.MessageFile = fmt.Sprintf("%s/%s", h.DataFileDir, dataFile.MessageFile)
	}

	h.Logger.Info("Creating temporary data")
	err := h.Db.CreateTemporaryData(dataFile)
	if err != nil {
		return err
	}
	return nil
}

func (h *Helper) ClearAllData() error {
	h.Logger.Info("Clearing all data")
	err := h.Db.ClearAllData()
	if err != nil {
		return err
	}
	return nil
}
