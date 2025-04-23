package server

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/api"
	"dungtl2003/chat-app-message-service/internal/config"
	ctx "dungtl2003/chat-app-message-service/internal/context"
	"dungtl2003/chat-app-message-service/internal/database"
	"dungtl2003/chat-app-message-service/internal/httpclient"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/router"
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

	// Create a new database connection
	db, err := database.New(config.DatabaseURL, loggerWrapper)
	if err != nil {
		loggerWrapper.Errorfln("database.New(): %v", err)
		os.Exit(1)
	}

	// Create a new validator
	validator := validate.NewValidator()

	// Create a new ID generator service
	idGeneratorService, err := snowflake.New(config.SnowflakeConfig.Addr, config.SnowflakeConfig.CertDir, loggerWrapper)
	if err != nil {
		loggerWrapper.Errorfln("snowflake.New(): %v", err)
		os.Exit(1)
	}

	client := httpclient.New()

	appCtx := &ctx.AppContext{
		IdGeneratorService: idGeneratorService,
		Validator:          validator,
		Logger:             loggerWrapper,
		Database:           db,
		Client:             client,
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
	// Run all services
	if err := s.appCtx.IdGeneratorService.Run(); err != nil {
		s.appCtx.Logger.Errorfln("IdGeneratorService.Run(): %v", err)
		os.Exit(1)
	}

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

// Close shuts down the server and all services. It also exits the program. It will log fatal if any error occurs.
func (s *Server) Close() {
	s.appCtx.Logger.Info("shutting down server...")

	// Inform the server it has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close all services
	if err := s.appCtx.IdGeneratorService.Close(); err != nil {
		s.appCtx.Logger.Errorfln("IdGeneratorService.Close(): %v", err)
		os.Exit(1)
	}

	// Close the database connection
	if err := s.appCtx.Database.Close(); err != nil {
		s.appCtx.Logger.Errorfln("Database.Close(): %v", err)
		os.Exit(1)
	}

	s.appCtx.Logger.Info("all services are closed")

	if err := s.srv.Shutdown(ctx); err != nil {
		s.appCtx.Logger.Errorfln("Shutdown(): %v", err)
		os.Exit(1)
	}

	s.appCtx.Logger.Info("server exiting")
	os.Exit(0)
}
