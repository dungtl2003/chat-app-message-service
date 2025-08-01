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

// New creates a new ID generator service instance and start it. The function
// will create a connection to the ID generator service. If the connection is
// successful, the function will return the service instance. Otherwise, the
// function will return an error. If the certDir is empty, the function will
// create an insecure connection. If the certDir is not empty, it must contain
// `client_cert.pem`, `client_key.pem`, and `ca_cert.pem`. The function will
// create a secure connection using these files. Remember to call Close() when
// done to release resources.
func New(serverAddr string, certDir string, logger *logging.LoggerWrapper) (*IdGeneratorService, error) {
	idGeneratorService := &IdGeneratorService{
		logger: logger,
	}
	opts := []grpc.DialOption{}

	if certDir != "" {
		logger.Infofln("[%s] Using TLS", idGeneratorService.Name())
		creds, err := loadTlsCredentials(certDir)
		if err != nil {
			logger.Errorfln("[%s] Failed to load TLS credentials: %v", idGeneratorService.Name(), err)
			return nil, err
		}

		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		logger.Infofln("[%s] Using insecure connection", idGeneratorService.Name())
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(serverAddr, opts...)
	if err != nil {
		logger.Errorfln("[%s] Failed to create connection, error: %v", idGeneratorService.Name(), err)
		return nil, err
	}

	idGeneratorService.conn = conn
	idGeneratorService.logger.Infofln("[%s] Connection established", idGeneratorService.Name())

	idGeneratorService.client = pb.NewIdGeneratorClient(conn)

	idGeneratorService.status = services.READY
	idGeneratorService.logger.Infofln("[%s] Running", idGeneratorService.Name())
	return idGeneratorService, nil
}

// Close closes the connection to the ID generator service.
func (s *IdGeneratorService) Close() error {
	if s.status == services.STOPPED {
		s.logger.Errorfln("[%s] Already stopped", s.Name())
		return nil
	}

	s.logger.Infofln("[%s] Closing", s.Name())
	err := s.conn.Close()
	if err != nil {
		s.logger.Errorfln("[%s] Failed to close connection, error: %v", s.Name(), err)
		s.status = services.ERROR
	}

	s.logger.Infofln("[%s] Stopped", s.Name())
	s.status = services.STOPPED

	return err
}

// GenerateId generates a new ID.
func (s *IdGeneratorService) GenerateId() (int64, error) {
	if s.status != services.READY {
		s.logger.Errorfln("[%s] ID generator service is not running", s.Name())
		return 0, fmt.Errorf("ID generator service is not running")
	}

	s.logger.Debugfln("[%s] Generating ID", s.Name())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.client.GenerateId(ctx, &pb.GenerateIdRequest{})
	if err != nil {
		s.status = services.ERROR
		s.logger.Errorfln("[%s] Failed to generate ID, error: %v", s.Name(), err)
		return 0, err
	}

	s.status = services.READY
	s.logger.Debugfln("[%s] Generated ID: %d", s.Name(), resp.Id)
	return resp.Id, nil
}

func (s *IdGeneratorService) Status() services.ServiceStatus {
	if s.status != services.STOPPED {
		_, err := s.GenerateId() // Check if the service is still ready by trying to generate an ID
		if err != nil {
			s.status = services.ERROR
			s.logger.Errorfln("[%s] Service is not ready, error: %v", s.Name(), err)
		} else {
			s.status = services.READY
		}
	}
	return s.status
}

func (s *IdGeneratorService) Name() string {
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
