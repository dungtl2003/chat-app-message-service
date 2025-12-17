package kafka

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/logging"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type DispatcherHandler struct {
	router  map[Topic]EventHandler
	logger  *logging.LoggerWrapper
	dlqChan chan<- KMessage[DLQEvent]
}

func (h *DispatcherHandler) Process(ctx context.Context, m kafka.Message) error {
	handler, exists := h.router[Topic(m.Topic)]
	if !exists {
		return fmt.Errorf("no handler registered for topic: %s", m.Topic)
	}

	h.logger.Debugfln("Dispatching message topic=%s key=%s", m.Topic, string(m.Key))
	return handler.Handle(ctx, m)
}

func (h *DispatcherHandler) MoveToDLQ(ctx context.Context, m kafka.Message, err error) {
	var sourceTopic Topic
	var dlqTopic Topic
	switch m.Topic {
	default:
		h.logger.Errorfln("[%s] Unknown topic for DLQ: %s", h.Name(), m.Topic)
		return
	}

	dlqEvent := DLQEvent{
		SourceTopic:   sourceTopic,
		OriginalKey:   m.Key,
		OriginalValue: m.Value,
		ErrorMessage:  err.Error(),
		Timestamp:     time.Now(),
	}
	dlqMessage := KMessage[DLQEvent]{
		Topic: dlqTopic,
		Key:   string(m.Key),
		Value: dlqEvent,
	}

	h.logger.Debugfln("[%s] Moving message to DLQ: %v", h.Name(), dlqMessage)
	h.dlqChan <- dlqMessage
}

func (h *DispatcherHandler) Name() string {
	return "Kafka Dispatcher Handler"
}
