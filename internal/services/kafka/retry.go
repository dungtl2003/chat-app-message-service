package kafka

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/logging"
	"io"
	"slices"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/segmentio/kafka-go"
)

type ProcessRetryHandler interface {
	Process(context.Context, kafka.Message) error
	MoveToDLQ(context.Context, kafka.Message, error)
}

type ConsumerWithRetryOptions struct {
	Handler              ProcessRetryHandler
	Reader               *kafka.Reader
	MaxRetries           int
	RetryQueue           chan kafka.Message
	NonRetryableTopics   []Topic // Topics that should not be retried
	Backoff              backoff.BackOff
	OnReadMessageFail    func(error)
	OnReadMessageSuccess func(kafka.Message)
}

// NewConsumerWithRetry creates a new kafka consumer with retry mechanism.
// Note that you should only use this function if the order of messages is not
// important.
func NewConsumerWithRetry(ctx context.Context, logger *logging.LoggerWrapper, options *ConsumerWithRetryOptions) {
	go func() {
		var lastError error
		lastError = nil
		for msg := range options.RetryQueue {
			for retries := 1; retries <= options.MaxRetries; {
				logger.Debugfln("Retrying message %v, attempt %d", msg.Key, retries)

				if retries >= options.MaxRetries {
					options.Handler.MoveToDLQ(ctx, msg, lastError)
					break
				}

				if err := options.Handler.Process(ctx, msg); err != nil {
					lastError = err
					logger.Errorfln("Error processing message: %v, retrying...", err)
					time.Sleep(options.Backoff.NextBackOff())
					retries++
					continue
				}
				break
			}
		}
	}()

	for {
		msg, err := options.Reader.FetchMessage(ctx)
		if err != nil {
			if err == io.EOF {
				logger.Infofln("EOF reached, stopping reading messages")
				return
			}

			logger.Errorfln("Error fetching message from Kafka: %v", err)
			if options.OnReadMessageFail != nil {
				options.OnReadMessageFail(err)
			}

			time.Sleep(1 * time.Second) // Wait before retrying to read
			continue
		}
		if err := options.Reader.CommitMessages(ctx, msg); err != nil {
			logger.Errorfln("Error committing message to Kafka: %v", err)
			if options.OnReadMessageFail != nil {
				options.OnReadMessageFail(err)
			}
			time.Sleep(1 * time.Second) // Wait before retrying to read
			continue
		}

		if options.OnReadMessageSuccess != nil {
			options.OnReadMessageSuccess(msg)
		}

		if err := options.Handler.Process(ctx, msg); err != nil {
			if options.NonRetryableTopics != nil && slices.Contains(options.NonRetryableTopics, Topic(msg.Topic)) {
				logger.Errorfln("Error processing message, moving to DLQ: %v", err)
				options.Handler.MoveToDLQ(ctx, msg, err)
			} else {
				logger.Errorfln("Error processing message, moving to retry queue: %v", err)
				options.RetryQueue <- msg
			}
		}
	}
}
