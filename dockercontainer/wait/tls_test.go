package wait_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"io"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/dockercontainer/wait"
)

const (
	serverName         = "127.0.0.1"
	caFilename         = "/tmp/ca.pem"
	clientCertFilename = "/tmp/cert.crt"
	clientKeyFilename  = "/tmp/cert.key"
)

var (
	//go:embed testdata/root.pem
	caBytes []byte

	//go:embed testdata/cert.crt
	certBytes []byte

	//go:embed testdata/cert.key
	keyBytes []byte
)

// testForTLSCert creates a new CertStrategy for testing.
func testForTLSCert() *wait.TLSStrategy {
	return wait.ForTLSCert(clientCertFilename, clientKeyFilename).
		WithRootCAs(caFilename).
		WithServerName(serverName).
		WithStartupTimeout(time.Millisecond * 50).
		WithPollInterval(time.Millisecond)
}

func TestForCert(t *testing.T) {
	errNotFound := errdefs.ErrNotFound.WithMessage("file not found")
	ctx := context.Background()

	t.Run("ca-not-found", func(t *testing.T) {
		target := newRunningTarget()
		target.EXPECT().CopyFromContainer(anyContext, caFilename).Return(nil, errNotFound)
		err := testForTLSCert().WaitUntilReady(ctx, target)
		require.EqualError(t, err, context.DeadlineExceeded.Error())
	})

	t.Run("cert-not-found", func(t *testing.T) {
		target := newRunningTarget()
		caFile := io.NopCloser(bytes.NewBuffer(caBytes))
		target.EXPECT().CopyFromContainer(anyContext, caFilename).Return(caFile, nil)
		target.EXPECT().CopyFromContainer(anyContext, clientCertFilename).Return(nil, errNotFound)
		err := testForTLSCert().WaitUntilReady(ctx, target)
		require.EqualError(t, err, context.DeadlineExceeded.Error())
	})

	t.Run("key-not-found", func(t *testing.T) {
		target := newRunningTarget()
		caFile := io.NopCloser(bytes.NewBuffer(caBytes))
		certFile := io.NopCloser(bytes.NewBuffer(certBytes))
		target.EXPECT().CopyFromContainer(anyContext, caFilename).Return(caFile, nil)
		target.EXPECT().CopyFromContainer(anyContext, clientCertFilename).Return(certFile, nil)
		target.EXPECT().CopyFromContainer(anyContext, clientKeyFilename).Return(nil, errNotFound)
		err := testForTLSCert().WaitUntilReady(ctx, target)
		require.EqualError(t, err, context.DeadlineExceeded.Error())
	})

	t.Run("valid", func(t *testing.T) {
		target := newRunningTarget()
		caFile := io.NopCloser(bytes.NewBuffer(caBytes))
		certFile := io.NopCloser(bytes.NewBuffer(certBytes))
		keyFile := io.NopCloser(bytes.NewBuffer(keyBytes))
		target.EXPECT().CopyFromContainer(anyContext, caFilename).Return(caFile, nil)
		target.EXPECT().CopyFromContainer(anyContext, clientCertFilename).Return(certFile, nil)
		target.EXPECT().CopyFromContainer(anyContext, clientKeyFilename).Return(keyFile, nil)

		certStrategy := testForTLSCert()
		err := certStrategy.WaitUntilReady(ctx, target)
		require.NoError(t, err)

		pool := x509.NewCertPool()
		require.True(t, pool.AppendCertsFromPEM(caBytes))
		cert, err := tls.X509KeyPair(certBytes, keyBytes)
		require.NoError(t, err)
		got := certStrategy.TLSConfig()
		require.Equal(t, serverName, got.ServerName)
		require.Equal(t, []tls.Certificate{cert}, got.Certificates)
		require.True(t, pool.Equal(got.RootCAs))
	})
}
