package server

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/api"
	"dungtl2003/chat-app-message-service/internal/config"
	ctx "dungtl2003/chat-app-message-service/internal/context"
	"dungtl2003/chat-app-message-service/internal/helper"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/router"
	"dungtl2003/chat-app-message-service/internal/services"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/idgen"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type MessageServerOptions struct {
	IdGeneratorService idgen.IdGeneratorService
}

type MessageServer struct {
	srv    *http.Server
	AppCtx *ctx.AppContext
}

// New creates a new MessageServer instance. The function will load the
// configuration and set up all necessary components. Call Run() to start the
// server. This function will return an error if there is an error when loading
// the configuration
func New(opts *MessageServerOptions) (*MessageServer, error) {
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

	loggerWrapper.Info("Creating application context")
	appCtx := &ctx.AppContext{
		IdGeneratorService: idGeneratorService,
		Validator:          validator,
		Logger:             loggerWrapper,
		DatabaseService:    databaseService,
		Services: []services.Service{
			idGeneratorService,
			databaseService,
		},
	}

	loggerWrapper.Info("Creating HTTP router")
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
		return nil, fmt.Errorf("error when creating router: %w", err)
	}

	loggerWrapper.Info("Creating HTTP server")
	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", config.ServerPort),
		Handler: router,
	}

	return &MessageServer{
		srv:    srv,
		AppCtx: appCtx,
	}, nil
}

// Run starts the server. The function will start the server and listen for signals
// to shut down the server. The function will exit the program if there is an error
// when starting the server. Call Close() to shut down the server.
func (s *MessageServer) Run() error {
	errSignal := make(chan error, 1)
	quit := make(chan os.Signal, 1)

	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.AppCtx.Logger.Error("error when starting server", "error", err)
			errSignal <- fmt.Errorf("error when starting server: %w", err)
		}
	}()
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errSignal:
		return err
	case <-quit:
		s.AppCtx.Logger.Info("received signal to shut down server")
		return s.Close()
	}
}

// Close shuts down the server. The function will close the database connection and
// shut down the server. The function will exit the program with status code 0 if
// the server is shut down successfully. The function will exit the program with
// status code 1 if there is an error when shutting down the server.
func (s *MessageServer) Close() error {
	s.AppCtx.Logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, service := range s.AppCtx.Services {
		if err := service.Close(); err != nil {
			s.AppCtx.Logger.Errorfln("error when closing service [%s]: %v", service.Name(), err)
			return err
		} else {
			s.AppCtx.Logger.Debugfln("service [%s] closed successfully", service.Name())
		}
	}

	if err := s.srv.Shutdown(ctx); err != nil {
		s.AppCtx.Logger.Errorfln("error when shutting down server: %v", err)
		return err
	} else {
		s.AppCtx.Logger.Infofln("server shut down")
	}

	return nil
}
