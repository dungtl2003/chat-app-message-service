package kafka

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/logging"
	"errors"
	"io"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/segmentio/kafka-go"
)

// ProcessingType defines how we handle the commit/ack flow
type ProcessingType int

const (
	// TypeSync: Process successfully FIRST, then Commit.
	// Used for strict ordering. Blocks fetching new messages until processed.
	TypeSync ProcessingType = iota

	// TypeAsyncRetryable: Commit IMMEDIATELY, then Process.
	// If failed, send to RetryQueue (background retry).
	TypeAsyncRetryable

	// TypeAsyncFireAndForget: Commit IMMEDIATELY, then Process.
	// If failed, send straight to DLQ (no retries).
	TypeAsyncFireAndForget
	// TypeSyncContinue: Process successfully FIRST, then Commit.
	// Used for strict ordering with limited retries.
	// Blocks fetching new messages until processed or retries exhausted.
	TypeSyncContinue
)

type TopicConfig struct {
	Type       ProcessingType
	MaxRetries int // Only used for Sync and AsyncRetryable
}

type ProcessRetryHandler interface {
	Process(context.Context, kafka.Message) error
	MoveToDLQ(context.Context, kafka.Message, error)
}

type ConsumerWithRetryOptions struct {
	Handler    ProcessRetryHandler
	Reader     *kafka.Reader
	RetryQueue chan kafka.Message
	Backoff    backoff.BackOff

	OnReadMessageFail    func(error)
	OnReadMessageSuccess func(kafka.Message)

	// Default config if topic is not found in the map
	DefaultTopicConfig TopicConfig
	// Configuration per topic
	TopicConfigs map[Topic]TopicConfig
}

// NewConsumerWithRetry creates a new kafka consumer with retry mechanism.
// Note that you should only use this function if the order of messages is not
// important.
func NewConsumerWithRetry(ctx context.Context, logger *logging.LoggerWrapper, options *ConsumerWithRetryOptions) {
	// Background Worker for ASYNC Retries
	// This consumes the RetryQueue channel. Note: These are only for "AsyncRetryable" messages.
	// "Sync" messages retry inside the main loop to preserve order.
	go func() {
		for msg := range options.RetryQueue {
			// Calculate retry strategy based on topic
			config := getTopicConfig(Topic(msg.Topic), options)

			// We use a simplified loop here because these are already "detached" from the stream
			processAsyncRetry(ctx, logger, options.Handler, msg, config.MaxRetries, options.Backoff)
		}
	}()

	// 2. Main Consumer Loop
	for {
		msg, err := options.Reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Infofln("EOF reached, stopping reader")
				return
			}
			logger.Errorfln("Error fetching message: %v", err)
			if options.OnReadMessageFail != nil {
				options.OnReadMessageFail(err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		if options.OnReadMessageSuccess != nil {
			options.OnReadMessageSuccess(msg)
		}

		// Determine strategy for this topic
		config := getTopicConfig(Topic(msg.Topic), options)

		switch config.Type {
		case TypeSync:
			handleSyncMessage(ctx, logger, options, msg, config)
		case TypeAsyncRetryable, TypeAsyncFireAndForget:
			handleAsyncMessage(ctx, logger, options, msg, config)
		}
	}
}

// getTopicConfig looks up the config or returns default
func getTopicConfig(topic Topic, opts *ConsumerWithRetryOptions) TopicConfig {
	if config, ok := opts.TopicConfigs[topic]; ok {
		return config
	}
	return opts.DefaultTopicConfig
}

// handleSyncMessage handles strict ordering.
func handleSyncMessage(
	ctx context.Context,
	logger *logging.LoggerWrapper,
	opts *ConsumerWithRetryOptions,
	msg kafka.Message,
	config TopicConfig,
) {
	var processErr error
	attempt := 0

	// Infinite Loop for Processing
	for {
		attempt++

		// 1. Try to Process
		processErr = opts.Handler.Process(ctx, msg)
		if processErr == nil {
			// Success! Break the loop to commit.
			break
		}

		// 2. Handle Failure
		// If we are in "Block" mode, we retry forever.
		// If we are in "Continue" mode, we check MaxRetries.
		if config.Type == TypeSyncContinue && attempt > config.MaxRetries {
			logger.Errorfln("Sync retries exhausted. Moving to DLQ: %v", processErr)
			opts.Handler.MoveToDLQ(ctx, msg, processErr)
			// We break here to allow the Commit to happen below (skipping the bad message)
			processErr = nil
			break
		}

		// 3. Log and Wait
		logger.Warnfln("[%s] Processing failed (attempt %d). Retrying... Error: %v",
			msg.Topic, attempt, processErr)

		// Backoff (capped at a reasonable max wait, e.g., 30s)
		sleepTime := min(opts.Backoff.NextBackOff(), 30*time.Second)

		select {
		case <-ctx.Done():
			// If server shuts down, we abort immediately.
			// We return WITHOUT committing. On restart, we will fetch this msg again.
			return
		case <-time.After(sleepTime):
			continue
		}
	}

	// 4. Commit (Only reachable if Success OR Moved to DLQ)
	if processErr == nil {
		if err := opts.Reader.CommitMessages(ctx, msg); err != nil {
			logger.Errorfln("Failed to commit sync message: %v", err)
		} else {
			logger.Debugfln("Message committed: %s", msg.Key)
		}
	}
}

// handleAsyncMessage handles high throughput.
// Flow: Fetch -> Commit -> Process -> (RetryQueue or DLQ)
func handleAsyncMessage(
	ctx context.Context,
	logger *logging.LoggerWrapper,
	opts *ConsumerWithRetryOptions,
	msg kafka.Message,
	config TopicConfig,
) {
	// Commit IMMEDIATELY (Ack right away)
	if err := opts.Reader.CommitMessages(ctx, msg); err != nil {
		logger.Errorfln("Failed to commit async message: %v", err)
		// Even if commit fails, we try to process because we already fetched it
	}

	// Process
	err := opts.Handler.Process(ctx, msg)
	if err == nil {
		return // Happy path
	}

	// Handle Failure
	if config.Type == TypeAsyncFireAndForget {
		// Fail Fast -> DLQ
		logger.Warnfln("Async (Fire&Forget) failed, moving to DLQ: %v", err)
		opts.Handler.MoveToDLQ(ctx, msg, err)
	} else {
		// Async Retryable -> Retry Queue
		logger.Warnfln("Async (Retryable) failed, moving to background retry queue: %v", err)

		// NON-BLOCKING send to channel (optional: select with default to avoid deadlocks if queue full)
		select {
		case opts.RetryQueue <- msg:
			// Sent to background worker
		default:
			logger.Errorfln("Retry queue full! Dropping message to DLQ: %v", err)
			opts.Handler.MoveToDLQ(ctx, msg, err)
		}
	}
}

// processAsyncRetry is the logic used by the background goroutine
func processAsyncRetry(
	ctx context.Context,
	logger *logging.LoggerWrapper,
	handler ProcessRetryHandler,
	msg kafka.Message,
	maxRetries int,
	bo backoff.BackOff,
) {
	var lastErr error

	// Note: We start at 1 because the first attempt (0) already happened in the main loop
	for i := 1; i <= maxRetries; i++ {
		logger.Debugfln("Background Retry %s (attempt %d/%d)", msg.Key, i, maxRetries)

		if err := handler.Process(ctx, msg); err != nil {
			lastErr = err
			time.Sleep(bo.NextBackOff())
			continue
		}
		return // Success
	}

	logger.Errorfln("Background retries exhausted. DLQ: %v", lastErr)
	handler.MoveToDLQ(ctx, msg, lastErr)
}
