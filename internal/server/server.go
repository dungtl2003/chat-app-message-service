package server

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/api"
	"dungtl2003/chat-app-message-service/internal/config"
	ctx "dungtl2003/chat-app-message-service/internal/context"
	"dungtl2003/chat-app-message-service/internal/httpclient"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/router"
	"dungtl2003/chat-app-message-service/internal/services"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/snowflake"
	"dungtl2003/chat-app-message-service/internal/validate"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	srv    *http.Server
	appCtx *ctx.AppContext
}

// New creates a new ConversationServer instance. It loads the configuration and creates a new database connection. It also creates a new validator and ID generator service. It returns the new instance. It will log fatal if any error occurs.
func New() *Server {
	log.Println("Loading configuration")
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("LoadConfig(): %v", err)
	}
	log.Printf("%s\n", config)

	logger, err := logging.NewLogger(config.LogConfig.Level, config.LogConfig.Kind)
	if err != nil {
		log.Fatalf("failed to create logger, error: %v", err)
	}
	loggerWrapper := logging.NewLoggerWrapper(logger)

	loggerWrapper.Info("switching to custom logger")

	// Create a new validator
	validator := validate.NewValidator()

	// Create a new ID generator service
	idGeneratorService, err := snowflake.New(config.SnowflakeConfig.Addr, config.SnowflakeConfig.CertDir, loggerWrapper)
	if err != nil {
		loggerWrapper.Errorfln("snowflake.New(): %v", err)
		os.Exit(1)
	}

	// Create a new database connection
	dbService, err := database.New(config.DatabaseURL, loggerWrapper)
	if err != nil {
		loggerWrapper.Errorfln("database.New(): %v", err)
		os.Exit(1)
	}

	client := httpclient.New()

	appCtx := &ctx.AppContext{
		IdGeneratorService: idGeneratorService,
		Validator:          validator,
		Logger:             loggerWrapper,
		DatabaseService:    dbService,
		Client:             client,
		Services: []services.Service{
			idGeneratorService,
			dbService,
		},
	}

	publicHandlers := []router.Handler{
		{
			Method: router.GET,
			Path:   "/healthcheck",
			H:      api.HealthCheck(appCtx),
		},
	}

	privateHandlers := []router.Handler{
		{
			Method: router.GET,
			Path:   "/conversations/:conversation-id/messages",
			H:      api.GetMessagesByConvID(appCtx),
		},

		{
			Method: router.POST,
			Path:   "/messages",
			H:      api.CreateMessage(appCtx),
		},
	}

	// Create a new router
	router, err := router.New(loggerWrapper, publicHandlers, privateHandlers)
	if err != nil {
		loggerWrapper.Errorfln("router.New(): %v", err)
		os.Exit(1)
	}
	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", config.ServerPort),
		Handler: router,
	}

	return &Server{
		srv:    srv,
		appCtx: appCtx,
	}
}

// Run starts the server and all services. It blocks until an error occurs or the server is closed. It will log fatal if any error occurs.
func (s *Server) Run() {
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.appCtx.Logger.Errorfln("ListenAndServe(): %v", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	s.Close()
}

// Close shuts down the server. The function will close the database connection and
// shut down the server. The function will exit the program with status code 0 if
// the server is shut down successfully. The function will exit the program with
// status code 1 if there is an error when shutting down the server.
func (s *Server) Close() {
	s.appCtx.Logger.Info("shutting down server")

	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
		if err != nil {
			os.Exit(1)
		}

		os.Exit(0)
	}()

	for _, service := range s.appCtx.Services {
		if err = service.Close(); err != nil {
			s.appCtx.Logger.Errorfln("error when closing service [%s]: %v", service.Name(), err)
		} else {
			s.appCtx.Logger.Debugfln("service [%s] closed successfully", service.Name())
		}
	}

	if err = s.srv.Shutdown(ctx); err != nil {
		s.appCtx.Logger.Errorfln("error when shutting down server: %v", err)
	} else {
		s.appCtx.Logger.Infofln("server shut down")
	}
}
