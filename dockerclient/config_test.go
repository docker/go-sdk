package dockerclient

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg, err := newConfig("docker-host")
		require.NoError(t, err)
		require.Equal(t, "docker-host", cfg.Host)
		require.False(t, cfg.TLSVerify)
		require.Empty(t, cfg.CertPath)
	})

	t.Run("success/tls-verify", func(t *testing.T) {
		certDir := filepath.Join("testdata", "certificates")

		t.Setenv("DOCKER_TLS_VERIFY", "1")
		t.Setenv("DOCKER_CERT_PATH", certDir)

		cfg, err := newConfig("docker-host")
		require.NoError(t, err)
		require.Equal(t, "docker-host", cfg.Host)
		require.True(t, cfg.TLSVerify)
		require.Equal(t, certDir, cfg.CertPath)
	})

	t.Run("error/host-required", func(t *testing.T) {
		cfg, err := newConfig("")
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("error/cert-path-required-for-tls", func(t *testing.T) {
		t.Setenv("DOCKER_TLS_VERIFY", "1")

		cfg, err := newConfig("docker-host")
		require.Error(t, err)
		require.Nil(t, cfg)
	})
}
