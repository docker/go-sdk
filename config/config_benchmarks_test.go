package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkAuthConfigCaching(b *testing.B) {
	cfg := Config{
		AuthConfigs: map[string]AuthConfig{
			"test.io": {Username: "user", Password: "pass"},
		},
	}

	b.Run("first-access", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg.clearAuthCache()
			_, err := cfg.AuthConfigForHostname("test.io")
			require.NoError(b, err)
		}
	})

	b.Run("cached-access", func(b *testing.B) {
		// Prime the cache
		_, _ = cfg.AuthConfigForHostname("test.io")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := cfg.AuthConfigForHostname("test.io")
			require.NoError(b, err)
		}
	})
}
