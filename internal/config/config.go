package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type LogConfig struct {
	Level string // INFO, DEBUG, WARN, ERROR
	Kind  string // TEXT or JSON
}

type SnowflakeConfig struct {
	Addr    string
	CertDir string
}

type Config struct {
	ServerPort      int
	Env             string
	LogConfig       *LogConfig
	DatabaseURL     string
	SnowflakeConfig *SnowflakeConfig
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
	err = c.setDatabaseURL()
	if err != nil {
		return nil, err
	}
	err = c.setSnowflakeConfig()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (l *LogConfig) String() string {
	parts := []string{
		fmt.Sprintf("LEVEL: %s", l.Level),
		fmt.Sprintf("KIND: %s", l.Kind),
	}

	return fmt.Sprintf("LogConfig{%s}", strings.Join(parts, ", "))
}

func (s *SnowflakeConfig) String() string {
	parts := []string{
		fmt.Sprintf("ADDR: %s", s.Addr),
		fmt.Sprintf("CERT_DIR: %s", s.CertDir),
	}

	return fmt.Sprintf("SnowflakeConfig{%s}", strings.Join(parts, ", "))
}

func (c *Config) String() string {
	parts := []string{
		fmt.Sprintf("SERVER_PORT: %d", c.ServerPort),
		fmt.Sprintf("LOG_CONFIG: %s", c.LogConfig),
		fmt.Sprintf("ENV: %s", c.Env),
		fmt.Sprintf("DATABASE_URL: %s", c.DatabaseURL),
		fmt.Sprintf("SNOWFLAKE_CONFIG: %s", c.SnowflakeConfig),
	}

	return fmt.Sprintf("Config{%s}", strings.Join(parts, ", "))
}

func (c *Config) setLogConfig() error {
	c.LogConfig = &LogConfig{}

	logLevel, has := os.LookupEnv("LOG_LEVEL")
	if !has {
		logLevel = "INFO"
	}
	if logLevel != "INFO" && logLevel != "DEBUG" && logLevel != "WARN" && logLevel != "ERROR" {
		return fmt.Errorf("`LOG_LEVEL=%s` is invalid. It can only be `INFO`, `DEBUG`, `WARN` or `ERROR`\n", logLevel)
	}

	kind, has := os.LookupEnv("LOG_KIND")
	if !has {
		kind = "TEXT"
	}

	if kind != "TEXT" && kind != "JSON" {
		return fmt.Errorf("`LOG_KIND=%s` is invalid, it can only be `TEXT` or `JSON`", kind)
	}

	c.LogConfig.Level = logLevel
	c.LogConfig.Kind = kind

	return nil
}

func (c *Config) setSnowflakeConfig() error {
	c.SnowflakeConfig = &SnowflakeConfig{}

	addr, has := os.LookupEnv("ID_GENERATOR_SERVICE_ADDR")
	if !has {
		return fmt.Errorf("ID_GENERATOR_SERVICE_ADDR not found")
	}

	certDir, has := os.LookupEnv("ID_GENERATOR_SERVICE_CERT_DIR")
	if !has {
		certDir = ""
	}

	c.SnowflakeConfig.Addr = addr
	c.SnowflakeConfig.CertDir = certDir

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
	env, has := os.LookupEnv("ENV")
	if !has {
		log.Println("ENV not found, setting to dev")
		env = "dev"
	}
	c.Env = env

	return nil
}

func (c *Config) setDatabaseURL() error {
	databaseURL, has := os.LookupEnv("DATABASE_URL")
	if !has {
		return fmt.Errorf("DATABASE_URL not found")
	}

	c.DatabaseURL = databaseURL
	return nil
}
