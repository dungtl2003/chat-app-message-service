package server

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/api"
	"dungtl2003/chat-app-message-service/internal/config"
	"dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/router"
	"dungtl2003/chat-app-message-service/internal/services"
	"dungtl2003/chat-app-message-service/internal/services/conversation"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/idgen"
	"dungtl2003/chat-app-message-service/internal/services/kafka"
	"dungtl2003/chat-app-message-service/internal/services/user"
	"dungtl2003/chat-app-message-service/internal/workers"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type MessageServerOptions struct {
	IdGeneratorService  idgen.IdGeneratorService
	ConversationService conversation.ConversationService
	UserService         user.UserService
}

type MessageServer struct {
	srv    *http.Server
	logger *logging.LoggerWrapper

	// Lifecycle management
	services []services.Service      // Things that need Close()
	workers  []func(context.Context) // Background tasks (Kafka consumers, etc)

	// Internal lifecycle management
	shutdownOnce sync.Once
	ctx          context.Context
	cancel       context.CancelFunc // To stop background workers
	wg           sync.WaitGroup     // To wait for background workers

}

// New creates a new MessageServer instance. The function will load the
// configuration and set up all necessary components. Call Run() to start the
// server. This function will return an error if there is an error when loading
// the configuration
func New(opts *MessageServerOptions) (*MessageServer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &MessageServer{
		services: make([]services.Service, 0),
		workers:  make([]func(context.Context), 0),
		cancel:   cancel,
		ctx:      ctx,
	}

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
	s.logger = loggerWrapper

	loggerWrapper.Info("Creating validator")
	validator := helper.NewValidator()

	loggerWrapper.Info("Creating database service")
	databaseService, err := database.New(config.DatabaseConfig.URL, loggerWrapper)
	if err != nil {
		return nil, fmt.Errorf("error when creating database service: %w", err)
	}

	var idGeneratorService idgen.IdGeneratorService
	if opts != nil && opts.IdGeneratorService != nil {
		loggerWrapper.Infofln("Using provided ID generator service")
		idGeneratorService = opts.IdGeneratorService
	} else {
		loggerWrapper.Infofln("Creating ID generator service")
		idGeneratorService, err = idgen.NewSnowflakeService(config.IdGeneratorConfig.Addr, &idgen.SnowflakeServiceOptions{
			Logger:  loggerWrapper,
			CertDir: config.IdGeneratorConfig.CertDir,
		})
		if err != nil {
			return nil, fmt.Errorf("error when creating ID generator service: %w", err)
		}
	}

	var userService user.UserService
	if opts != nil && opts.UserService != nil {
		loggerWrapper.Infofln("Using provided user service")
		userService = opts.UserService
	} else {
		loggerWrapper.Infofln("Creating user service")
		userService, err = user.NewUserServiceV1(
			config.UserConfig.URL,
			&user.UserServiceV1Options{
				Logger: loggerWrapper,
			})
		if err != nil {
			return nil, fmt.Errorf("error when creating user service: %w", err)
		}
	}

	var conversationService conversation.ConversationService
	if opts != nil && opts.ConversationService != nil {
		loggerWrapper.Infofln("Using provided conversation service")
		conversationService = opts.ConversationService
	} else {
		loggerWrapper.Infofln("Creating conversation service")
		conversationService, err = conversation.NewConversationServiceV1(
			config.ConversationConfig.URL,
			&conversation.ConversationServiceV1Options{
				Logger: loggerWrapper,
			})
		if err != nil {
			return nil, fmt.Errorf("error when creating conversation service: %w", err)
		}
	}

	dlqChan := make(chan kafka.KMessage[kafka.DLQEvent], 100)
	msgChan := make(chan kafka.KMessage[kafka.MessageResourceCreatedEvent], 100)
	assetConfirmChan := make(chan kafka.KMessage[kafka.AssetResourceConfirmEvent], 100)

	loggerWrapper.Infofln("Creating Kafka producer service")
	kafkaProducerService, err := kafka.NewKafkaProducer(
		config.KafkaConfig.Brokers,
		loggerWrapper,
	)
	if err != nil {
		return nil, fmt.Errorf("error when creating Kafka writer service: %w", err)
	}

	loggerWrapper.Infofln("Setting up background workers")
	s.workers = append(s.workers, func(ctx context.Context) {
		forwarder := &workers.Forwarder[kafka.DLQEvent]{
			Producer: kafkaProducerService,
			Source:   dlqChan,
			Logger:   loggerWrapper,
		}
		forwarder.Start(ctx)
	})
	s.workers = append(s.workers, func(ctx context.Context) {
		forwarder := &workers.Forwarder[kafka.MessageResourceCreatedEvent]{
			Producer: kafkaProducerService,
			Source:   msgChan,
			Logger:   loggerWrapper,
		}
		forwarder.Start(ctx)
	})
	s.workers = append(s.workers, func(ctx context.Context) {
		forwarder := &workers.Forwarder[kafka.AssetResourceConfirmEvent]{
			Producer: kafkaProducerService,
			Source:   assetConfirmChan,
			Logger:   loggerWrapper,
		}
		forwarder.Start(ctx)
	})
	s.workers = append(s.workers, func(ctx context.Context) {
		processor := &workers.OutboxProcessor{
			DB:                  databaseService,
			Logger:              loggerWrapper,
			OutboxCheckInterval: config.OutboxProcessorConfig.CheckInterval,
			Producer:            kafkaProducerService,
		}
		processor.Start(ctx)
	})

	loggerWrapper.Info("Creating application context")
	handlerDeps := &api.HandlerDeps{
		IdGeneratorService:       idGeneratorService,
		Validator:                validator,
		Logger:                   loggerWrapper,
		DatabaseService:          databaseService,
		Config:                   config,
		UserService:              userService,
		ConversationService:      conversationService,
		AssetConfirmEventChannel: assetConfirmChan,
	}

	services := []services.Service{
		idGeneratorService,
		databaseService,
		kafkaProducerService,
		userService,
		conversationService,
	}
	s.services = services

	loggerWrapper.Info("Creating HTTP router")
	publicHandlers := []router.Handler{
		{
			Method: router.GET,
			Path:   "/healthcheck",
			H:      api.HealthCheck(handlerDeps),
		},
	}
	privateHandlers := []router.Handler{
		{
			Method: router.GET,
			Path:   "/conversations/:conversation-id/messages",
			H:      api.GetMessagesByConvID(handlerDeps),
		},
		{
			// get message by ID
			Method: router.GET,
			Path:   "/messages/:message-id",
			H:      api.GetMessageByID(handlerDeps),
		},
		{
			Method: router.POST,
			Path:   "/messages",
			H:      api.CreateMessage(handlerDeps),
		},
	}
	// Create a new router
	router, err := router.New(loggerWrapper, publicHandlers, privateHandlers)
	if err != nil {
		return nil, fmt.Errorf("error when creating router: %w", err)
	}

	loggerWrapper.Info("Creating HTTP server")
	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", config.ServerPort),
		Handler: router,
	}
	s.srv = srv

	return s, nil
}

// Run starts the server. It listens for incoming HTTP requests and handles
// them according to the defined routes. Remember to call Close() to shut down
// the server gracefully.
func (s *MessageServer) Run() error {
	// Start background workers
	for _, w := range s.workers {
		worker := w
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			worker(s.ctx)
		}()
	}

	// Start HTTP server
	go func() {
		s.logger.Infofln("HTTP server listening on %s", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Errorfln("HTTP server error: %v", err)
			s.cancel() // stop workers if server fails
		}
	}()

	// Wait for OS signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	s.logger.Info("Shutdown signal received")

	return s.Close()
}

// Close gracefully shuts down the HTTP server and releases all resources.
func (s *MessageServer) Close() error {
	var finalErr error

	// Ensure we only close once
	s.shutdownOnce.Do(func() {
		s.logger.Info("Starting graceful shutdown sequence...")

		// Stop Background Workers
		s.logger.Debug("Stopping background workers...")
		s.cancel()  // Cancel the context passed to workers
		s.wg.Wait() // Wait for them to finish their current task

		// Shutdown HTTP Server
		s.logger.Debug("Shutting down HTTP server...")

		// Create a timeout context specifically for the shutdown procedure
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Errorfln("HTTP shutdown error: %v", err)
			// We don't return immediately; we still want to close DB/Kafka
			finalErr = err
		}

		// Close External Resources (DB, Kafka, etc.)
		s.logger.Debug("Closing external services...")
		for _, service := range s.services {
			if err := service.Close(); err != nil {
				s.logger.Errorfln("Error closing service [%s]: %v", service.Name(), err)
				if finalErr == nil {
					finalErr = err
				}
			} else {
				s.logger.Debugfln("Service [%s] closed", service.Name())
			}
		}

		// Close channels if strictly necessary (usually not needed if writers are stopped)

		s.logger.Info("Server shutdown complete.")
	})

	return finalErr
}
