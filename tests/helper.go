package tests

import (
	"dungtl2003/chat-app-message-service/internal/config"
	"dungtl2003/chat-app-message-service/internal/database"
	"dungtl2003/chat-app-message-service/internal/httpclient"
	"dungtl2003/chat-app-message-service/internal/logging"
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
	Db              *database.Database
	Client          *httpclient.HttpClient
	Logger          *logging.LoggerWrapper
	SnowflakeConfig *SnowflakeConfig
}

func NewHelper() *Helper {
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config.LoadConfig(): %v", err)
	}

	logger, err := logging.NewLogger(config.LogConfig.Level, config.LogConfig.Kind)
	if err != nil {
		log.Fatalf("logging.NewLogger(): %v", err)
	}
	loggerWrapper := logging.NewLoggerWrapper(logger)
	if err != nil {
		log.Fatalf("logging.NewLoggerWrapper(): %v", err)
	}

	dbURL, has := os.LookupEnv("ADMIN_DATABASE_URL")
	if !has {
		log.Fatalf("Error when getting ADMIN_DATABASE_URL")
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
	}

	return helper
}
