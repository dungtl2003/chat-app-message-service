package config

import (
	"dungtl2003/chat-app-message-service/internal/logging"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type LogConfig struct {
	Level logging.LoggerLevel
	Kind  logging.LoggerKind
}

type IdGeneratorConfig struct {
	Addr    string
	CertDir string
	Epoch   int64
}

type DatabaseConfig struct {
	URL     string
	ReadURL string
}

type MediaConfig struct {
	URL string
}

type UserConfig struct {
	URL string
}

type ConversationConfig struct {
	URL string
}

type KafkaConfig struct {
	Brokers []string
}

type OutboxProcessorConfig struct {
	CheckInterval time.Duration
}

type Config struct {
	ServerPort            int
	Env                   string
	LogConfig             LogConfig
	IdGeneratorConfig     IdGeneratorConfig
	DatabaseConfig        DatabaseConfig
	MediaConfig           MediaConfig
	UserConfig            UserConfig
	ConversationConfig    ConversationConfig
	KafkaConfig           KafkaConfig
	OutboxProcessorConfig OutboxProcessorConfig
}

// LoadConfig loads the configuration from env file. It will return Config instance
// or error if occurs.
func LoadConfig() (*Config, error) {
	c := &Config{}
	err := c.setEnv()
	if err != nil {
		return nil, err
	}
	err = c.setServerPort()
	if err != nil {
		return nil, err
	}
	err = c.setLogConfig()
	if err != nil {
		return nil, err
	}
	err = c.setDatabaseConfig()
	if err != nil {
		return nil, err
	}
	err = c.setIdGeneratorConfig()
	if err != nil {
		return nil, err
	}
	err = c.setMediaConfig()
	if err != nil {
		return nil, err
	}
	err = c.setUserConfig()
	if err != nil {
		return nil, err
	}
	err = c.setConversationConfig()
	if err != nil {
		return nil, err
	}
	err = c.setKafkaConfig()
	if err != nil {
		return nil, err
	}
	err = c.setOutboxProcessorConfig()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (k KafkaConfig) String() string {
	parts := []string{
		fmt.Sprintf("BROKERS: %s", strings.Join(k.Brokers, ", ")),
	}

	return fmt.Sprintf("KafkaConfig{%s}", strings.Join(parts, ", "))
}

func (o OutboxProcessorConfig) String() string {
	parts := []string{
		fmt.Sprintf("CHECK_INTERVAL: %s", o.CheckInterval),
	}

	return fmt.Sprintf("OutboxProcessorConfig{%s}", strings.Join(parts, ", "))
}

func (l LogConfig) String() string {
	parts := []string{
		fmt.Sprintf("LEVEL: %s", l.Level),
		fmt.Sprintf("KIND: %s", l.Kind),
	}

	return fmt.Sprintf("LogConfig{%s}", strings.Join(parts, ", "))
}

func (m MediaConfig) String() string {
	return fmt.Sprintf("MediaConfig{URL: %s}", m.URL)
}

func (u UserConfig) String() string {
	return fmt.Sprintf("UserConfig{URL: %s}", u.URL)
}

func (c ConversationConfig) String() string {
	return fmt.Sprintf("ConversationConfig{URL: %s}", c.URL)
}

func (d DatabaseConfig) String() string {
	parts := []string{
		fmt.Sprintf("URL: %s", d.URL),
		fmt.Sprintf("READ_URL: %s", d.ReadURL),
	}

	return fmt.Sprintf("DatabaseConfig{%s}", strings.Join(parts, ", "))
}

func (s IdGeneratorConfig) String() string {
	parts := []string{
		fmt.Sprintf("ADDR: %s", s.Addr),
		fmt.Sprintf("CERT_DIR: %s", s.CertDir),
		fmt.Sprintf("EPOCH: %d", s.Epoch),
	}

	return fmt.Sprintf("SnowflakeConfig{%s}", strings.Join(parts, ", "))
}

func (c Config) String() string {
	parts := []string{
		fmt.Sprintf("SERVER_PORT: %d", c.ServerPort),
		fmt.Sprintf("LOG_CONFIG: %s", c.LogConfig),
		fmt.Sprintf("ENV: %s", c.Env),
		fmt.Sprintf("DATABASE_CONFIG: %s", c.DatabaseConfig),
		fmt.Sprintf("ID_GENERATOR_CONFIG: %s", c.IdGeneratorConfig),
		fmt.Sprintf("MEDIA_CONFIG: %s", c.MediaConfig),
		fmt.Sprintf("USER_CONFIG: %s", c.UserConfig),
		fmt.Sprintf("CONVERSATION_CONFIG: %s", c.ConversationConfig),
		fmt.Sprintf("KAFKA_CONFIG: %s", c.KafkaConfig),
		fmt.Sprintf("OUTBOX_PROCESSOR_CONFIG: %s", c.OutboxProcessorConfig),
	}

	return fmt.Sprintf("Config{%s}", strings.Join(parts, ", "))
}

func (c *Config) setKafkaConfig() error {
	log.Println("Setting KAFKA_BROKERS")
	brokersStr, has := os.LookupEnv("KAFKA_BROKERS")
	if !has {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	brokers := strings.Split(brokersStr, ",")
	if len(brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS must contain at least one broker")
	}
	c.KafkaConfig.Brokers = brokers
	return nil
}

func (c *Config) setOutboxProcessorConfig() error {
	intervalStr, has := os.LookupEnv("OUTBOX_CHECK_INTERVAL_MS")
	outboxCheckIntervalMs := 500 // Default to 500ms
	if has {
		interval, err := strconv.Atoi(intervalStr)
		if err != nil {
			return fmt.Errorf("OUTBOX_CHECK_INTERVAL_MS is invalid: %v", err)
		}
		outboxCheckIntervalMs = interval
	}

	c.OutboxProcessorConfig.CheckInterval = time.Duration(outboxCheckIntervalMs) * time.Millisecond
	return nil
}

func (c *Config) setLogConfig() error {
	logLevel := logging.INFO // Default log level
	logKind := logging.TEXT  // Default log kind

	logLevelStr, has := os.LookupEnv("LOG_LEVEL")
	if has {
		if !logging.IsValidLoggerLevel(logLevelStr) {
			return fmt.Errorf("`LOG_LEVEL=%s` is invalid", logLevelStr)
		}
		logLevel = logging.LoggerLevel(logLevelStr)
	}

	kind, has := os.LookupEnv("LOG_KIND")
	if has {
		if !logging.IsValidLoggerKind(kind) {
			return fmt.Errorf("`LOG_KIND=%s` is invalid", kind)
		}
		logKind = logging.LoggerKind(kind)
	}

	c.LogConfig.Level = logLevel
	c.LogConfig.Kind = logKind
	return nil
}

func (c *Config) setMediaConfig() error {
	mediaUrl, has := os.LookupEnv("MEDIA_SERVICE_URL")
	if !has {
		return fmt.Errorf("MEDIA_SERVICE_URL not found")
	}

	c.MediaConfig.URL = mediaUrl

	return nil
}

func (c *Config) setUserConfig() error {
	userUrl, has := os.LookupEnv("USER_SERVICE_URL")
	if !has {
		return fmt.Errorf("USER_SERVICE_URL not found")
	}

	c.UserConfig.URL = userUrl

	return nil
}

func (c *Config) setConversationConfig() error {
	conversationUrl, has := os.LookupEnv("CONVERSATION_SERVICE_URL")
	if !has {
		return fmt.Errorf("CONVERSATION_SERVICE_URL not found")
	}

	c.ConversationConfig.URL = conversationUrl

	return nil
}

func (c *Config) setIdGeneratorConfig() error {
	addr, has := os.LookupEnv("ID_GENERATOR_ADDR")
	if !has {
		return fmt.Errorf("ID_GENERATOR_ADDR not found")
	}

	certDir, has := os.LookupEnv("ID_GENERATOR_CERT_DIR")
	if !has {
		certDir = ""
	}

	epochStr, has := os.LookupEnv("ID_GENERATOR_EPOCH")
	var epoch int64 = 1672531200000
	if has {
		var err error
		epoch, err = strconv.ParseInt(epochStr, 10, 64)
		if err != nil {
			return fmt.Errorf("ID_GENERATOR_EPOCH is invalid: %v", err)
		}
	}

	c.IdGeneratorConfig.Epoch = epoch
	c.IdGeneratorConfig.Addr = addr
	c.IdGeneratorConfig.CertDir = certDir

	return nil
}

func (c *Config) setServerPort() error {
	portStr, has := os.LookupEnv("PORT")
	if !has {
		log.Println("PORT not found, setting to 8400")
		portStr = "8400"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("Port number out of range: %s (1-65535)", portStr)
	}

	c.ServerPort = port
	return nil
}

func (c *Config) setEnv() error {
	log.Println("Setting ENVIRONMENT")
	env, has := os.LookupEnv("ENVIRONMENT")
	if !has {
		log.Println("ENVIRONMENT not found, setting to dev")
		env = "dev"
	}
	c.Env = env

	return nil
}

func (c *Config) setDatabaseConfig() error {
	log.Println("Setting DATABASE_URL")
	url, has := os.LookupEnv("DATABASE_URL")
	if !has {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if url == "" {
		return fmt.Errorf("DATABASE_URL cannot be empty")
	}

	readUrl, has := os.LookupEnv("DATABASE_READ_URL")
	if has && readUrl != "" {
		c.DatabaseConfig.ReadURL = readUrl
	} else {
		c.DatabaseConfig.ReadURL = url
	}

	c.DatabaseConfig.URL = url
	return nil
}
