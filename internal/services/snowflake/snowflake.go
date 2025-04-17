package snowflake

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services"
	pb "dungtl2003/chat-app-message-service/internal/services/snowflake/proto"
	"fmt"
	"os"
	"path"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	CLIENT_CERT_FILE = "client_cert.pem"
	CLIENT_KEY_FILE  = "client_key.pem"
	SERVER_CA_FILE   = "ca_cert.pem"
)

type IdGeneratorService struct {
	status services.ServiceStatus
	client pb.IdGeneratorClient
	conn   *grpc.ClientConn
	logger *logging.LoggerWrapper
}

// New creates a new ID generator service instance. The function will create a
// connection to the ID generator service. If the connection is successful, the
// function will return the service instance. Otherwise, the function will return
// an error. If the certDir is empty, the function will create an insecure connection.
// If the certDir is not empty, it must contain `client_cert.pem`, `client_key.pem`,
// and `ca_cert.pem`. The function will create a secure connection using these files.
// Call Run() to start the service.
func New(serverAddr string, certDir string, logger *logging.LoggerWrapper) (*IdGeneratorService, error) {
	idGeneratorService := &IdGeneratorService{
		logger: logger,
	}
	opts := []grpc.DialOption{}

	if certDir != "" {
		logger.Info("using TLS", "service", idGeneratorService.GetName())
		creds, err := loadTlsCredentials(certDir)
		if err != nil {
			logger.Error("failed to load TLS credentials", "service", idGeneratorService.GetName(), "error", err)
			return nil, err
		}

		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		logger.Info("using insecure connection", "service", idGeneratorService.GetName())
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(serverAddr, opts...)
	if err != nil {
		errMessage := fmt.Sprintf("failed to create connection to ID generator service, error: %v", err)
		logger.Error(errMessage, "service", idGeneratorService.GetName())
		return nil, err
	}

	idGeneratorService.conn = conn
	idGeneratorService.status = services.READY
	idGeneratorService.logger.Info("ID generator service is ready", "service", idGeneratorService.GetName())
	return idGeneratorService, nil
}

// Run starts the ID generator service.
func (s *IdGeneratorService) Run() error {
	client := pb.NewIdGeneratorClient(s.conn)
	s.client = client

	s.status = services.RUNNING
	s.logger.Info("ID generator service is running", "service", s.GetName())
	return nil
}

// Close closes the connection to the ID generator service.
func (s *IdGeneratorService) Close() error {
	if s.status == services.STOPPED {
		s.logger.Info("ID generator service is already stopped", "service", s.GetName())
		return nil
	}

	s.logger.Info("closing ID generator service", "service", s.GetName())
	err := s.conn.Close()
	if err != nil {
		s.logger.Error("failed to close connection to ID generator service", "service", s.GetName(), "error", err)
		s.status = services.ERROR
	}

	s.logger.Info("ID generator service is closed", "service", s.GetName())
	s.status = services.STOPPED

	return err
}

// GenerateId generates a new ID.
func (s *IdGeneratorService) GenerateId() (int64, error) {
	if s.status != services.RUNNING {
		s.logger.Error("ID generator service is not running", "service", s.GetName(), "status", s.GetStatus())
		return 0, fmt.Errorf("ID generator service is not running")
	}

	s.logger.Info("generating ID", "service", s.GetName())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.client.GenerateId(ctx, &pb.GenerateIdRequest{})
	if err != nil {
		s.logger.Error("failed to generate ID", "service", s.GetName(), "error", err)
		return 0, err
	}

	s.logger.Info("generated ID", "service", s.GetName(), "id", resp.Id)
	return resp.Id, nil
}

func (s *IdGeneratorService) GetStatus() services.ServiceStatus {
	return s.status
}

func (s *IdGeneratorService) GetName() string {
	return "ID generator"
}

func loadTlsCredentials(certDir string) (credentials.TransportCredentials, error) {
	var (
		clientCertPath = path.Join(certDir, CLIENT_CERT_FILE)
		clientKeyPath  = path.Join(certDir, CLIENT_KEY_FILE)
		serverCaPath   = path.Join(certDir, SERVER_CA_FILE)
	)

	if _, err := os.Stat(clientCertPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", clientCertPath)
	}

	if _, err := os.Stat(clientKeyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", clientCertPath)
	}

	if _, err := os.Stat(serverCaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", clientCertPath)
	}

	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, err
	}

	pemCA, err := os.ReadFile(serverCaPath)
	if err != nil {
		return nil, err
	}

	certPools := x509.NewCertPool()
	if !certPools.AppendCertsFromPEM(pemCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate, path: %s", serverCaPath)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPools,
	}
	return credentials.NewTLS(config), nil
}
