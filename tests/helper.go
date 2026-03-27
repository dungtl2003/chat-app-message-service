package tests

import (
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/server"
	"dungtl2003/chat-app-message-service/internal/services/database"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	KAFKA_GROUP_ID = "chat-app-message-service-test-group"
)

type SnowflakeConfig struct {
	Addr    string
	CertDir string
}

type IdGeneratorConfig struct {
	TLSAddr     string
	NonTLSAddr  string
	CertDir     string
	FakeCertDir string
}

type TestHelper struct {
	AdminDatabaseService *database.DatabaseService
	Client               *http.Client
	Logger               *logging.LoggerWrapper
	IdGeneratorConfig    IdGeneratorConfig
	DataFileDir          string
	MessageServiceURL    string
	BrokerAddr           string

	server *server.MessageServer
}

type SetUpOptions struct {
	DataFile      *database.DataFile
	ServerOptions *server.MessageServerOptions
}

func NewTestHelper() *TestHelper {
	brokerAddr, bool := os.LookupEnv("BROKER_ADDR")
	if !bool {
		log.Fatal("BROKER_ADDR is not set")
	}

	logger, err := logging.NewLogger(logging.DEBUG, logging.TEXT)
	if err != nil {
		log.Fatalf("Error when loading logger: %v", err)
	}
	loggerWrapper := logging.NewLoggerWrapper(logger)

	loggerWrapper.Info("Starting test helper setup")

	messageServiceURL, bool := os.LookupEnv("MESSAGE_SERVICE_URL")
	if !bool {
		log.Fatal("MESSAGE_SERVICE_URL is not set")
	}

	dbURL, bool := os.LookupEnv("ADMIN_DATABASE_URL")
	if !bool {
		log.Fatal("ADMIN_DATABASE_URL is not set")
	}

	db, err := database.New(dbURL, dbURL, loggerWrapper)
	if err != nil {
		log.Fatalf("Error when creating database connection: %v", err)
	}

	loggerWrapper.Info("Creating http client")
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: 15 * time.Second,
		},
		Timeout: 15 * time.Second,
	}

	dataFileDir, bool := os.LookupEnv("DATA_FILE_DIR")
	if !bool {
		log.Fatal("DATA_FILE_DIR is not set")
	}

	snowflakeConfig, err := getIdGeneratorConfig()
	if err != nil {
		log.Fatalf("Error when getting snowflake config: %v", err)
	}

	h := &TestHelper{
		BrokerAddr:           brokerAddr,
		AdminDatabaseService: db,
		Client:               client,
		Logger:               loggerWrapper,
		MessageServiceURL:    messageServiceURL,
		IdGeneratorConfig:    snowflakeConfig,
		DataFileDir:          dataFileDir,
	}

	h.Logger.Info("Test helper setup completed successfully")
	return h
}

func SetUp(t *TestHelper, opts *SetUpOptions) {
	if err := t.clearAllData(); err != nil {
		log.Fatalf("Error when clearing all data: %v", err)
	}

	if opts != nil && opts.DataFile != nil {
		t.Logger.Info("Creating test data")
		if err := t.createTestData(*opts.DataFile); err != nil {
			log.Fatalf("Error when creating test data: %v", err)
		}
	} else {
		t.Logger.Info("No test data provided, skipping data creation")
	}

	var serverOpts *server.MessageServerOptions
	serverOpts = nil
	if opts != nil {
		serverOpts = opts.ServerOptions
	}

	server, err := server.New(serverOpts)
	if err != nil {
		log.Fatalf("Error when creating server: %v", err)
	}
	t.server = server

	t.Logger.Info("Starting server")
	go func() {
		err := server.Run()
		if err != nil {
			log.Fatalf("Error when running server: %v", err)
		}
	}()

	err = waitForServer(fmt.Sprintf("%s/healthcheck", t.MessageServiceURL), 5*time.Second)
	if err != nil {
		t.Logger.Errorfln("Server did not start in time: %v", err)
		log.Fatalf("Error waiting for server to start: %v", err)
	}
}

func (h *TestHelper) createTestData(dataFile database.DataFile) error {
	if dataFile.UserFile != "" {
		dataFile.UserFile = fmt.Sprintf("%s/%s", h.DataFileDir, dataFile.UserFile)
	}
	if dataFile.ConversationFile != "" {
		dataFile.ConversationFile = fmt.Sprintf("%s/%s", h.DataFileDir, dataFile.ConversationFile)
	}
	if dataFile.MessageFile != "" {
		dataFile.MessageFile = fmt.Sprintf("%s/%s", h.DataFileDir, dataFile.MessageFile)
	}
	if dataFile.AssetFile != "" {
		dataFile.AssetFile = fmt.Sprintf("%s/%s", h.DataFileDir, dataFile.AssetFile)
	}

	h.Logger.Info("Creating temporary data")
	err := h.AdminDatabaseService.CreateTemporaryData(dataFile)
	if err != nil {
		return err
	}
	return nil
}

func TearDown(t *TestHelper) {
	t.Logger.Info("Tearing down test helper")

	t.Logger.Info("Clearing all data")
	if err := t.clearAllData(); err != nil {
		log.Fatalf("Error when clearing all data: %v", err)
	}

	t.Logger.Info("Closing admin database service")
	if err := t.AdminDatabaseService.Close(); err != nil {
		log.Fatalf("Error when closing admin database service: %v", err)
	}

	t.Logger.Info("Closing server")
	err := t.server.Close()
	if err != nil {
		log.Fatalf("Error when closing server: %v", err)
	}

	t.Logger.Info("Test helper torn down successfully")
}

func Get(client *http.Client, url string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header = header
	return client.Do(req)
}

func Put(client *http.Client, url string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}

	req.Header = header
	return client.Do(req)
}

func Post(client *http.Client, url string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	if header != nil {
		req.Header = header
	}
	return client.Do(req)
}

func Patch(client *http.Client, url string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PATCH", url, body)
	if err != nil {
		return nil, err
	}

	req.Header = header
	return client.Do(req)
}

func Delete(client *http.Client, url string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header = header
	return client.Do(req)
}

// GetInternalAccessToken returns a mock access token for a given user ID. If the user ID
// is not found in the mapping, it returns an empty string.
func GetInternalAccessToken(userId int64) string {
	mapping := map[int64]string{
		2:                   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXAiOiJKV1QiLCJzdWIiOiJ1c2VyOjIiLCJhdWQiOlsiaW50ZXJuYWwtc2VydmljZSJdLCJpYXQiOjE3MDY4NzA0MDAsImV4cCI6MTcwNjg3NDAwMH0.xQz5Ytlyuibsdcuh3LB-uNeFnK_DCkjKdw7eG0gy51g",
		3:                   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXAiOiJKV1QiLCJzdWIiOiJ1c2VyOjMiLCJhdWQiOlsiaW50ZXJuYWwtc2VydmljZSJdLCJpYXQiOjE3MDY4NzA0MDAsImV4cCI6MTcwNjg3NDAwMH0.PrpCyxaHe0s9YJnT6CZ08449QIWc5XA7PQgZB9VUm5Q",
		4:                   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXAiOiJKV1QiLCJzdWIiOiJ1c2VyOjQiLCJhdWQiOlsiaW50ZXJuYWwtc2VydmljZSJdLCJpYXQiOjE3MDY4NzA0MDAsImV4cCI6MTcwNjg3NDAwMH0.V2EMlSIlEu1K6hRMSFcsBEsPBBF52OhSSFyB8oGIx4Q",
		5:                   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXAiOiJKV1QiLCJzdWIiOiJ1c2VyOjUiLCJhdWQiOlsiaW50ZXJuYWwtc2VydmljZSJdLCJpYXQiOjE3MDY4NzA0MDAsImV4cCI6MTcwNjg3NDAwMH0.V2EMlSIlEu1K6hRMSFcsBEsPBBF52OhSSFyB8oGIx4Q",
		6:                   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXAiOiJKV1QiLCJzdWIiOiJ1c2VyOjYiLCJhdWQiOlsiaW50ZXJuYWwtc2VydmljZSJdLCJpYXQiOjE3MDY4NzA0MDAsImV4cCI6MTcwNjg3NDAwMH0.V2EMlSIlEu1K6hRMSFcsBEsPBBF52OhSSFyB8oGIx4Q",
		12345:               "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXAiOiJKV1QiLCJzdWIiOiJ1c2VyOjEyMzQ1IiwiYXVkIjpbImludGVybmFsLXNlcnZpY2UiXSwiaWF0IjoxNzA2ODcwNDAwLCJleHAiOjE3MDY4NzQwMDB9.V2EMlSIlEu1K6hRMSFcsBEsPBBF52OhSSFyB8oGIx4Q",
		9223372036854775800: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXAiOiJKV1QiLCJzdWIiOiJ1c2VyOjkyMjMzNzIwMzY4NTQ3NzU4MDAiLCJhdWQiOlsiaW50ZXJuYWwtc2VydmljZSJdLCJpYXQiOjE3MDY4NzA0MDAsImV4cCI6MTcwNjg3NDAwMH0.V2EMlSIlEu1K6hRMSFcsBEsPBBF52OhSSFyB8oGIx4Q",
	}

	if token, exists := mapping[userId]; exists {
		return token
	}

	return ""
}

func (t *TestHelper) clearAllData() error {
	t.Logger.Info("Clearing all data")
	if err := t.AdminDatabaseService.ClearAllData(); err != nil {
		t.Logger.Errorfln("Failed to clear all data: %v", err)
		return err
	}

	return nil
}

func getIdGeneratorConfig() (IdGeneratorConfig, error) {
	tlsAddr, ok := os.LookupEnv("ID_GENERATOR_TLS_ADDR")
	if !ok {
		return IdGeneratorConfig{}, fmt.Errorf("ID_GENERATOR_TLS_ADDR is not set")
	}
	nonTlsAddr, ok := os.LookupEnv("ID_GENERATOR_NON_TLS_ADDR")
	if !ok {
		return IdGeneratorConfig{}, fmt.Errorf("ID_GENERATOR_NON_TLS_ADDR is not set")
	}
	certDir, ok := os.LookupEnv("ID_GENERATOR_CERT_DIR")
	if !ok {
		return IdGeneratorConfig{}, fmt.Errorf("ID_GENERATOR_CERT_DIR is not set")
	}
	fakeCertDir, ok := os.LookupEnv("ID_GENERATOR_FAKE_CERT_DIR")
	if !ok {
		return IdGeneratorConfig{}, fmt.Errorf("ID_GENERATOR_FAKE_CERT_DIR is not set")
	}

	return IdGeneratorConfig{
		TLSAddr:     tlsAddr,
		NonTLSAddr:  nonTlsAddr,
		CertDir:     certDir,
		FakeCertDir: fakeCertDir,
	}, nil
}

func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server did not start at %s within %s", url, timeout)
}
