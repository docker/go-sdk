package dockerconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkLoadConfig(b *testing.B) {
	tmpDir := b.TempDir()

	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte(`{
		"auths": {
			"https://index.docker.io/v1/": {
				"auth": "dGVzdHVzZXI6dGVzdHBhc3N3b3Jk"
			},
			"https://registry.example.com": {
				"auth": "YW5vdGhlcnVzZXI6YW5vdGhlcnBhc3N3b3Jk"
			}
		},
		"credHelpers": {
			"registry.example.com": "ecr-login"
		}
	}`), 0o644)
	require.NoError(b, err)

	b.Setenv("DOCKER_CONFIG", tmpDir)

	b.Run("load-default", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			cfg, err := Load()
			require.NoError(b, err)
			require.NotNil(b, cfg)
		}
	})

	b.Run("load-with-auth-config", func(b *testing.B) {
		authConfig := `{
			"auths": {
				"https://index.docker.io/v1/": {
					"auth": "dGVzdHVzZXI6dGVzdHBhc3N3b3Jk"
				}
			}
		}`
		b.Setenv("DOCKER_AUTH_CONFIG", authConfig)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			cfg, err := Load()
			require.NoError(b, err)
			require.NotNil(b, cfg)
		}
	})
}

func BenchmarkGetCredentials(b *testing.B) {
	tmpDir := b.TempDir()

	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte(`{
		"auths": {
			"https://index.docker.io/v1/": {
				"auth": "dGVzdHVzZXI6dGVzdHBhc3N3b3Jk"
			},
			"https://registry.example.com": {
				"auth": "YW5vdGhlcnVzZXI6YW5vdGhlcnBhc3N3b3Jk"
			}
		},
		"credHelpers": {
			"registry.example.com": "ecr-login"
		}
	}`), 0o644)
	require.NoError(b, err)

	b.Setenv("DOCKER_CONFIG", tmpDir)

	// Load config once for reuse
	cfg, err := Load()
	require.NoError(b, err)
	require.NotNil(b, cfg)

	b.Run("docker-io", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			username, password, err := cfg.RegistryCredentialsForHostname("https://index.docker.io/v1/")
			require.NoError(b, err)
			require.Equal(b, "testuser", username)
			require.Equal(b, "testpassword", password)
		}
	})

	b.Run("example-com", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			username, password, err := cfg.RegistryCredentialsForHostname("https://registry.example.com")
			require.NoError(b, err)
			require.Equal(b, "anotheruser", username)
			require.Equal(b, "anotherpassword", password)
		}
	})

	b.Run("not-found", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			username, password, err := cfg.RegistryCredentialsForHostname("https://nonexistent.registry.com")
			require.NoError(b, err)
			require.Empty(b, username)
			require.Empty(b, password)
		}
	})

	b.Run("registry-credentials-for-image", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			username, password, err := RegistryCredentials("docker.io/library/nginx:latest")
			require.NoError(b, err)
			require.Equal(b, "testuser", username)
			require.Equal(b, "testpassword", password)
		}
	})
}
