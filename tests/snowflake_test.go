package tests

import (
	"context"
	"dungtl2003/chat-app-message-service/internal/services/idgen"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Connection test for ID generator service
// It should work with both tls and non-tls connection
func TestSnowflakeServiceConnection(t *testing.T) {
	helper := NewTestHelper()
	SetUp(helper, nil)
	defer TearDown(helper)

	nonTlsServerAddr := helper.IdGeneratorConfig.NonTLSAddr
	tlsServerAddr := helper.IdGeneratorConfig.TLSAddr
	certDir := helper.IdGeneratorConfig.CertDir

	snowflakeServiceNonTls, err := idgen.NewSnowflakeService(nonTlsServerAddr, &idgen.SnowflakeServiceOptions{
		Logger: helper.Logger,
	})
	require.NoError(t, err, "failed to create snowflake service (non TLS)")
	defer snowflakeServiceNonTls.Close()

	snowflakeServiceTls, err := idgen.NewSnowflakeService(tlsServerAddr, &idgen.SnowflakeServiceOptions{
		CertDir: certDir,
		Logger:  helper.Logger,
	})
	require.NoError(t, err, "failed to create snowflake service (TLS)")
	defer snowflakeServiceTls.Close()

	for _, srv := range []idgen.IdGeneratorService{snowflakeServiceTls, snowflakeServiceNonTls} {
		success := false
		attempts := 10
		for i := range attempts {
			helper.Logger.Debugfln("Attempt %d to generate ID from snowflake service", i+1)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err = srv.GenerateId(ctx)
			if err == nil {
				success = true
				break
			}
			helper.Logger.Errorfln("failed to generate ID from snowflake service: %v", err)
		}

		require.Truef(t, success, "failed to generate ID from snowflake service after %d attempts", attempts)
	}
}

func TestSnowflakeServiceConnectionWithFakeCert(t *testing.T) {
	helper := NewTestHelper()
	SetUp(helper, nil)
	defer TearDown(helper)

	tlsServerAddr := helper.IdGeneratorConfig.TLSAddr
	fakeCertDir := helper.IdGeneratorConfig.FakeCertDir

	snowflakeServiceFakeCert, err := idgen.NewSnowflakeService(tlsServerAddr, &idgen.SnowflakeServiceOptions{
		CertDir: fakeCertDir,
		Logger:  helper.Logger,
	})
	if err != nil {
		return
	}
	defer snowflakeServiceFakeCert.Close()

	attempts := 10
	for i := range attempts {
		helper.Logger.Debugfln("Attempt %d to generate ID from snowflake service with fake cert", i+1)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = snowflakeServiceFakeCert.GenerateId(ctx)
		require.Error(t, err, "expected to fail generating ID from ID generator service with fake cert")
	}
}
