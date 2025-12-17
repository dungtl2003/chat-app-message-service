package workers

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services/kafka"
)

// Forwarder connects a typed channel to the generic KafkaProducer
type Forwarder[T kafka.KEvent] struct {
	Producer *kafka.KafkaProducer
	Source   <-chan kafka.KMessage[T]
	Logger   *logging.LoggerWrapper
}

// Start consumes the channel and writes to Kafka until the channel is closed
func (f *Forwarder[T]) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-f.Source:
			if !ok {
				f.Logger.Infofln("[%s] Channel closed, stopping forwarder", f.Name())
				return
			}

			// Call the generic writer
			err := f.Producer.Write(ctx, string(msg.Topic), msg.Key, msg.Value)
			if err != nil {
				// Handle retry logic or critical failure here if needed
				// Note: kafka-go writer handles retries internally for network issues
				f.Logger.Errorfln("[%s] Drop message due to write error: %v", f.Name(), err)
			}
		}
	}
}

func (f *Forwarder[T]) Name() string {
	return "Kafka Forwarder"
}
