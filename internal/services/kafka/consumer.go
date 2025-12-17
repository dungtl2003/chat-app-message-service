package kafka

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/segmentio/kafka-go"
)

type ReaderConfig struct {
	Brokers            []string
	GroupID            string
	MinBytes           int
	MaxBytes           int
	NonRetryableTopics []Topic
}

type KafkaConsumerService struct {
	handlers map[Topic]EventHandler // The Router Map
	readers  []*kafka.Reader
	logger   *logging.LoggerWrapper
	config   ReaderConfig

	// Dependencies for Retry/DLQ
	retryChan chan kafka.Message
	dlqChan   chan<- KMessage[DLQEvent]

	// Lifecycle management
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewKafkaConsumerService(
	config ReaderConfig,
	logger *logging.LoggerWrapper,
	dlqChan chan<- KMessage[DLQEvent],
	retryChan chan kafka.Message,
) *KafkaConsumerService {
	return &KafkaConsumerService{
		handlers:  make(map[Topic]EventHandler),
		logger:    logger,
		config:    config,
		dlqChan:   dlqChan,
		retryChan: retryChan,
	}
}

// RegisterHandler binds a topic to a specific business logic handler
func (k *KafkaConsumerService) RegisterHandler(topic Topic, handler EventHandler) {
	k.handlers[topic] = handler
}

// Start initializes readers for all registered topics and starts consuming
func (k *KafkaConsumerService) Start(ctx context.Context) error {
	ctx, k.cancel = context.WithCancel(ctx)

	if len(k.config.Brokers) == 0 {
		return fmt.Errorf("no brokers provided")
	}

	// Initialize a reader for each registered topic
	for topic := range k.handlers {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:         k.config.Brokers,
			GroupID:         k.config.GroupID,
			Topic:           string(topic),
			MinBytes:        k.config.MinBytes,
			MaxBytes:        k.config.MaxBytes,
			ReadLagInterval: 15 * time.Second,
		})
		k.readers = append(k.readers, r)

		k.wg.Add(1)
		go k.consumeLoop(ctx, r)
	}

	k.logger.Infofln("Kafka consumer started for topics: %v", k.getRegisteredTopics())
	return nil
}

func (k *KafkaConsumerService) consumeLoop(ctx context.Context, r *kafka.Reader) {
	defer k.wg.Done()

	// We create a "RetryHandler" wrapper here that implements the Process interface
	// expected by your 'NewConsumerWithRetry' logic
	retryHandler := &DispatcherHandler{
		router:  k.handlers, // Pass the map
		logger:  k.logger,
		dlqChan: k.dlqChan,
	}

	topicConfigs := make(map[Topic]TopicConfig)
	topicConfigs[MESSAGE_RESOURCE_CREATED_TOPIC] = TopicConfig{
		Type: TypeSync,
	}
	NewConsumerWithRetry(ctx, k.logger, &ConsumerWithRetryOptions{
		Handler:      retryHandler, // The dispatcher is the handler
		Reader:       r,
		RetryQueue:   k.retryChan,
		Backoff:      backoff.NewExponentialBackOff(),
		TopicConfigs: topicConfigs,
		OnReadMessageFail: func(err error) {
			k.logger.Errorfln("Read error: %v", err)
		},
	})
}

func (k *KafkaConsumerService) Close() error {
	k.cancel() // Stop contexts
	for _, r := range k.readers {
		r.Close()
	}
	k.wg.Wait() // Wait for routines to finish
	return nil
}

func (k *KafkaConsumerService) Name() string {
	return "Kafka Consumer Service"
}

func (k *KafkaConsumerService) Status() services.ServiceStatus {
	// For simplicity, we assume it's always ready if started
	return services.ServiceReady
}

func (k *KafkaConsumerService) getRegisteredTopics() []Topic {
	keys := make([]Topic, 0, len(k.handlers))
	for k := range k.handlers {
		keys = append(keys, k)
	}
	return keys
}
