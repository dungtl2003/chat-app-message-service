package workers

import (
	"context"
	"database/sql"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/model"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"dungtl2003/chat-app-message-service/internal/services/kafka"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type OutboxProcessor struct {
	DB                  *database.DatabaseService
	Logger              *logging.LoggerWrapper
	OutboxCheckInterval time.Duration
	Producer            *kafka.KafkaProducer
}

// Start runs the processor continuously until the context is cancelled.
func (w *OutboxProcessor) Start(ctx context.Context) {
	w.Logger.Infofln("[%s] Starting...", w.Name())

	// Ticker controls how often we check for new events when the DB is empty
	ticker := time.NewTicker(w.OutboxCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.Logger.Infofln("[%s] Stopping...", w.Name())
			return
		case <-ticker.C:
			w.processNextBatch(ctx)
		}
	}
}

func (w *OutboxProcessor) processNextBatch(ctx context.Context) {
	// Fetch a batch of outboxEvents
	// We fetch a small batch (e.g., 50) to keep memory low and lock times short
	outboxEvents, err := w.DB.FetchPendingOutboxEvents(ctx, 50)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			w.Logger.Errorfln("[%s] Failed to fetch outbox events: %v", w.Name(), err)
		}
		return
	}

	if len(outboxEvents) == 0 {
		w.Logger.Debugfln("[%s] No pending outbox events found", w.Name())
		return // Yield back to ticker
	}

	var wg sync.WaitGroup

	// Process concurrently?
	// Since we used DISTINCT ON (conversation_id), every event in this list
	// belongs to a DIFFERENT conversation. Therefore, it is safe to process
	// this specific batch in parallel goroutines if you want high throughput!

	for _, outbox := range outboxEvents {
		kafkaEvent, err := toKafkaMessageEvent(outbox)
		if err != nil {
			w.Logger.Errorfln("[%s] Failed to convert outbox event %d to Kafka event: %v",
				w.Name(), outbox.Id.Int64(), err)
			continue
		}

		wg.Add(1)
		go func(e kafka.KMessage[kafka.MessageResourceCreatedEvent]) {
			defer wg.Done()
			w.processSingleEvent(ctx, e)
		}(kafkaEvent)
	}

	wg.Wait()
}

func (w *OutboxProcessor) processSingleEvent(ctx context.Context, event kafka.KMessage[kafka.MessageResourceCreatedEvent]) {
	// Send to Kafka
	// We rely on the Producer's synchronous Write or internal delivery guarantee.
	// Ensure w.Producer is injected into this struct.
	err := w.Producer.Write(ctx, string(event.Topic), event.Key, event.Value)
	payload := event.Value

	if err != nil {
		w.Logger.Errorfln("[%s] Failed to publish event (outboxId: %d, convEventId: %d, convId: %d): %v",
			w.Name(), payload.OutboxId.Int64(), payload.ConversationEventId.Int64(), payload.ConversationId.Int64(), err)

		// Handle Failure
		// This unlocks the row and sets a future retry time.
		if dbErr := w.DB.MarkOutboxEventFailed(ctx, payload.OutboxId.Int64(), err); dbErr != nil {
			w.Logger.Errorfln("[%s] Failed to mark outbox event %d as failed: %v", w.Name(), payload.OutboxId.Int64(), dbErr)
		}
		return
	}

	// Handle Success
	if dbErr := w.DB.MarkOutboxEventSent(ctx, payload.OutboxId.Int64()); dbErr != nil {
		// Critical error: Message sent to Kafka but DB failed to update.
		// This is why consumers must be Idempotent.
		w.Logger.Errorfln("[%s] CRITICAL: Message %d sent but DB update failed: %v", w.Name(), payload.Message.Id.Int64(), dbErr)
	} else {
		w.Logger.Debugfln("[%s] Successfully processed outbox event %d", w.Name(), payload.OutboxId.Int64())
	}
}

func (w *OutboxProcessor) Name() string {
	return "Outbox Processor"
}

func toKafkaMessageEvent(m model.MessageOutbox) (kafka.KMessage[kafka.MessageResourceCreatedEvent], error) {
	payload := model.MessageOutboxPayload{}
	err := json.Unmarshal([]byte(m.Payload.RawMessage), &payload)
	if err != nil {
		// Handle error appropriately in real implementation
		return kafka.KMessage[kafka.MessageResourceCreatedEvent]{}, err
	}
	return kafka.KMessage[kafka.MessageResourceCreatedEvent]{
		Topic: kafka.MESSAGE_RESOURCE_CREATED_TOPIC,
		Key:   m.ConversationId.String(),
		Value: kafka.MessageResourceCreatedEvent{
			Message:             payload.Message,
			ConversationEventId: m.ConversationEventId,
			ConversationId:      m.ConversationId,
			OutboxId:            m.Id,
		},
	}, nil
}
