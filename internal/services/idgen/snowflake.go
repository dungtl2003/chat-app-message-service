package idgen

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"dungtl2003/chat-app-message-service/internal/logging"
	"dungtl2003/chat-app-message-service/internal/services"
	"fmt"
	"os"
	"path"
	"time"

	pb "dungtl2003/chat-app-message-service/internal/services/idgen/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	CLIENT_CERT_FILE = "client_cert.pem"
	CLIENT_KEY_FILE  = "client_key.pem"
	SERVER_CA_FILE   = "ca_cert.pem"
)

type SnowflakeService struct {
	status             services.ServiceStatus
	client             pb.IdGeneratorClient
	conn               *grpc.ClientConn
	logger             *logging.LoggerWrapper
	healthCheckTimeout time.Duration
}

type SnowflakeServiceOptions struct {
	HealthCheckTimeout time.Duration // Timeout for health check requests
	Logger             *logging.LoggerWrapper
	CertDir            string // Directory containing TLS certificates
}

// NewSnowflakeService creates a new ID generator service instance and start it. The function
// will create a connection to the ID generator service. If the connection is
// successful, the function will return the service instance. Otherwise, the
// function will return an error. If the certDir is empty, the function will
// create an insecure connection. If the certDir is not empty, it must contain
// `client_cert.pem`, `client_key.pem`, and `ca_cert.pem`. The function will
// create a secure connection using these files. Remember to call Close() when
// done to release resources.
func NewSnowflakeService(serverAddr string, opts *SnowflakeServiceOptions) (*SnowflakeService, error) {
	var loggerWrapper *logging.LoggerWrapper

	if opts != nil && opts.Logger == nil {
		logger, err := logging.NewLogger(logging.INFO, logging.TEXT)
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %v", err)
		}
		loggerWrapper = logging.NewLoggerWrapper(logger)
	} else {
		loggerWrapper = opts.Logger
	}

	snowflakeService := &SnowflakeService{
		logger: loggerWrapper,
	}

	grpcOpts := []grpc.DialOption{}

	if opts != nil && opts.CertDir != "" {
		loggerWrapper.Infofln("[%s] Using TLS", snowflakeService.Name())
		creds, err := loadTlsCredentials(opts.CertDir)
		if err != nil {
			loggerWrapper.Errorfln("[%s] Failed to load TLS credentials: %v", snowflakeService.Name(), err)
			return nil, err
		}

		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(creds))
	} else {
		loggerWrapper.Infofln("[%s] Using insecure connection", snowflakeService.Name())
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if opts != nil && opts.HealthCheckTimeout > 0 {
		snowflakeService.healthCheckTimeout = opts.HealthCheckTimeout
	} else {
		snowflakeService.healthCheckTimeout = 5 * time.Second // Default health check timeout
	}

	conn, err := grpc.NewClient(serverAddr, grpcOpts...)
	if err != nil {
		loggerWrapper.Errorfln("[%s] Failed to create connection, error: %v", snowflakeService.Name(), err)
		return nil, err
	}

	snowflakeService.conn = conn
	snowflakeService.logger.Infofln("[%s] Connection established", snowflakeService.Name())

	snowflakeService.client = pb.NewIdGeneratorClient(conn)

	snowflakeService.status = services.READY
	snowflakeService.logger.Infofln("[%s] Running", snowflakeService.Name())
	return snowflakeService, nil
}

// Close closes the connection to the ID generator service.
func (s *SnowflakeService) Close() error {
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
func (s *SnowflakeService) GenerateId(ctx context.Context) (int64, error) {
	if s.status == services.STOPPED {
		return 0, fmt.Errorf("service is stopped")
	}
	if s.status != services.READY {
		s.logger.Warnfln("[%s] Service is not ready", s.Name())
	}

	s.logger.Debugfln("[%s] Generating ID", s.Name())
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

func (s *SnowflakeService) Status() services.ServiceStatus {
	if s.status != services.STOPPED {
		ctx, cancel := context.WithTimeout(context.Background(), s.healthCheckTimeout)
		defer cancel()

		s.status = services.READY   // GenerateId() needs service to be READY to proceed
		_, err := s.GenerateId(ctx) // Check if the service is still ready by trying to generate an ID
		if err != nil {
			s.status = services.ERROR
			s.logger.Errorfln("[%s] Service is not ready, error: %v", s.Name(), err)
		} else {
			s.status = services.READY
		}
	}
	return s.status
}

func (s *SnowflakeService) Name() string {
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
