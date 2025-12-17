package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// EventHandler defines the interface for processing a specific topic
type EventHandler interface {
	Handle(ctx context.Context, msg kafka.Message) error
}

// EventHandlerFunc allows us to use simple functions as handlers
type EventHandlerFunc func(ctx context.Context, msg kafka.Message) error

func (f EventHandlerFunc) Handle(ctx context.Context, msg kafka.Message) error {
	return f(ctx, msg)
}
