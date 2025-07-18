package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/go-sdk/config"
)

func TestToRegistryAuthConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    config.AuthConfig
		expected registry.AuthConfig
	}{
		{
			name:  "empty config",
			input: config.AuthConfig{},
			expected: registry.AuthConfig{
				Email: "",
			},
		},
		{
			name: "basic username and password",
			input: config.AuthConfig{
				Username: "testuser",
				Password: "testpass",
			},
			expected: registry.AuthConfig{
				Username: "testuser",
				Password: "testpass",
				Email:    "",
			},
		},
		{
			name: "with auth field",
			input: config.AuthConfig{
				Username: "user",
				Password: "pass",
				Auth:     "dXNlcjpwYXNz", // base64 encoded "user:pass"
			},
			expected: registry.AuthConfig{
				Username: "user",
				Password: "pass",
				Auth:     "dXNlcjpwYXNz",
				Email:    "",
			},
		},
		{
			name: "with server address",
			input: config.AuthConfig{
				Username:      "user",
				Password:      "pass",
				ServerAddress: "registry.example.com",
			},
			expected: registry.AuthConfig{
				Username:      "user",
				Password:      "pass",
				Email:         "",
				ServerAddress: "registry.example.com",
			},
		},
		{
			name: "with identity token",
			input: config.AuthConfig{
				Username:      "user",
				IdentityToken: "identity-token-123",
			},
			expected: registry.AuthConfig{
				Username:      "user",
				Email:         "",
				IdentityToken: "identity-token-123",
			},
		},
		{
			name: "with registry token",
			input: config.AuthConfig{
				Username:      "user",
				RegistryToken: "registry-token-456",
			},
			expected: registry.AuthConfig{
				Username:      "user",
				Email:         "",
				RegistryToken: "registry-token-456",
			},
		},
		{
			name: "complete config",
			input: config.AuthConfig{
				Username:      "testuser",
				Password:      "testpass",
				Auth:          "dGVzdHVzZXI6dGVzdHBhc3M=", // base64 encoded "testuser:testpass"
				ServerAddress: "registry.example.com",
				IdentityToken: "identity-token-123",
				RegistryToken: "registry-token-456",
			},
			expected: registry.AuthConfig{
				Username:      "testuser",
				Password:      "testpass",
				Auth:          "dGVzdHVzZXI6dGVzdHBhc3M=",
				Email:         "",
				ServerAddress: "registry.example.com",
				IdentityToken: "identity-token-123",
				RegistryToken: "registry-token-456",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ToRegistryAuthConfig(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestToRegistryAuthConfig_EmailAlwaysEmpty(t *testing.T) {
	// Test that Email field is always set to empty string, regardless of input
	input := config.AuthConfig{
		Username: "user",
		Password: "pass",
	}

	result := ToRegistryAuthConfig(input)
	require.Empty(t, result.Email, "Email field should always be empty")
}

func TestConfigToConfigFile(t *testing.T) {
	t.Helper()
	tests := []struct {
		name     string
		input    config.Config
		validate func(t *testing.T, result *configfile.ConfigFile)
	}{
		{
			name:  "empty config",
			input: config.Config{},
			validate: func(t *testing.T, result *configfile.ConfigFile) {
				t.Helper()
				require.NotNil(t, result)
				require.NotNil(t, result.AuthConfigs)
				require.NotNil(t, result.Plugins)
				require.NotNil(t, result.Features)
				require.Equal(t, map[string]types.AuthConfig{}, result.AuthConfigs)
				require.Equal(t, map[string]map[string]string{}, result.Plugins)
				require.Equal(t, map[string]string{}, result.Features)
			},
		},
		{
			name: "basic config with auth",
			input: config.Config{
				AuthConfigs: map[string]config.AuthConfig{
					"registry.example.com": {
						Username: "testuser",
						Password: "testpass",
					},
				},
				HTTPHeaders: map[string]string{
					"User-Agent": "test-client",
				},
				PsFormat:         "table {{.ID}}",
				ImagesFormat:     "table {{.Repository}}",
				CredentialsStore: "desktop",
				CurrentContext:   "test-context",
			},
			validate: func(t *testing.T, result *configfile.ConfigFile) {
				t.Helper()
				require.NotNil(t, result)
				require.Equal(t, "testuser", result.AuthConfigs["registry.example.com"].Username)
				require.Equal(t, "testpass", result.AuthConfigs["registry.example.com"].Password)
				require.Empty(t, result.AuthConfigs["registry.example.com"].Email)
				require.Equal(t, "test-client", result.HTTPHeaders["User-Agent"])
				require.Equal(t, "table {{.ID}}", result.PsFormat)
				require.Equal(t, "table {{.Repository}}", result.ImagesFormat)
				require.Equal(t, "desktop", result.CredentialsStore)
				require.Equal(t, "test-context", result.CurrentContext)
			},
		},
		{
			name: "config with proxies",
			input: config.Config{
				Proxies: map[string]config.ProxyConfig{
					"default": {
						HTTPProxy:  "http://proxy.example.com:8080",
						HTTPSProxy: "https://proxy.example.com:8443",
						NoProxy:    "localhost,127.0.0.1",
						FTPProxy:   "ftp://proxy.example.com:21",
					},
				},
			},
			validate: func(t *testing.T, result *configfile.ConfigFile) {
				t.Helper()
				require.NotNil(t, result)
				require.Contains(t, result.Proxies, "default")
				proxy := result.Proxies["default"]
				require.Equal(t, "http://proxy.example.com:8080", proxy.HTTPProxy)
				require.Equal(t, "https://proxy.example.com:8443", proxy.HTTPSProxy)
				require.Equal(t, "localhost,127.0.0.1", proxy.NoProxy)
				require.Equal(t, "ftp://proxy.example.com:21", proxy.FTPProxy)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ToConfigFile(tc.input)
			tc.validate(t, result)
		})
	}
}

func TestToCLIAuthConfig(t *testing.T) {
	t.Helper()
	tests := []struct {
		name     string
		input    config.AuthConfig
		expected types.AuthConfig
	}{
		{
			name:  "empty config",
			input: config.AuthConfig{},
			expected: types.AuthConfig{
				Email: "",
			},
		},
		{
			name: "basic username and password",
			input: config.AuthConfig{
				Username: "testuser",
				Password: "testpass",
			},
			expected: types.AuthConfig{
				Username: "testuser",
				Password: "testpass",
				Email:    "",
			},
		},
		{
			name: "with auth field",
			input: config.AuthConfig{
				Username: "user",
				Password: "pass",
				Auth:     "dXNlcjpwYXNz",
			},
			expected: types.AuthConfig{
				Username: "user",
				Password: "pass",
				Auth:     "dXNlcjpwYXNz",
				Email:    "",
			},
		},
		{
			name: "with server address and identity token",
			input: config.AuthConfig{
				Username:      "user",
				ServerAddress: "registry.example.com",
				IdentityToken: "identity-token-123",
			},
			expected: types.AuthConfig{
				Username:      "user",
				Email:         "",
				ServerAddress: "registry.example.com",
				IdentityToken: "identity-token-123",
			},
		},
		{
			name: "with registry token included",
			input: config.AuthConfig{
				Username:      "user",
				Password:      "pass",
				RegistryToken: "registry-token-456",
			},
			expected: types.AuthConfig{
				Username:      "user",
				Password:      "pass",
				Email:         "",
				RegistryToken: "registry-token-456",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ToCLIAuthConfig(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestToCLIAuthConfigs(t *testing.T) {
	t.Helper()
	tests := []struct {
		name     string
		input    map[string]config.AuthConfig
		expected map[string]types.AuthConfig
	}{
		{
			name:     "empty map",
			input:    map[string]config.AuthConfig{},
			expected: map[string]types.AuthConfig{},
		},
		{
			name: "single auth config",
			input: map[string]config.AuthConfig{
				"registry.example.com": {
					Username: "testuser",
					Password: "testpass",
				},
			},
			expected: map[string]types.AuthConfig{
				"registry.example.com": {
					Username: "testuser",
					Password: "testpass",
					Email:    "",
				},
			},
		},
		{
			name: "multiple auth configs",
			input: map[string]config.AuthConfig{
				"registry1.example.com": {
					Username: "user1",
					Password: "pass1",
					Auth:     "dXNlcjE6cGFzczE=",
				},
				"registry2.example.com": {
					Username:      "user2",
					IdentityToken: "token123",
				},
			},
			expected: map[string]types.AuthConfig{
				"registry1.example.com": {
					Username: "user1",
					Password: "pass1",
					Auth:     "dXNlcjE6cGFzczE=",
					Email:    "",
				},
				"registry2.example.com": {
					Username:      "user2",
					IdentityToken: "token123",
					Email:         "",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ToCLIAuthConfigs(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestToCLIProxyConfig(t *testing.T) {
	t.Helper()
	tests := []struct {
		name     string
		input    config.ProxyConfig
		expected configfile.ProxyConfig
	}{
		{
			name:     "empty config",
			input:    config.ProxyConfig{},
			expected: configfile.ProxyConfig{},
		},
		{
			name: "basic proxy config",
			input: config.ProxyConfig{
				HTTPProxy:  "http://proxy.example.com:8080",
				HTTPSProxy: "https://proxy.example.com:8443",
				NoProxy:    "localhost,127.0.0.1",
			},
			expected: configfile.ProxyConfig{
				HTTPProxy:  "http://proxy.example.com:8080",
				HTTPSProxy: "https://proxy.example.com:8443",
				NoProxy:    "localhost,127.0.0.1",
			},
		},
		{
			name: "complete proxy config",
			input: config.ProxyConfig{
				HTTPProxy:  "http://proxy.example.com:8080",
				HTTPSProxy: "https://proxy.example.com:8443",
				FTPProxy:   "ftp://proxy.example.com:21",
				NoProxy:    "localhost,127.0.0.1,.internal",
			},
			expected: configfile.ProxyConfig{
				HTTPProxy:  "http://proxy.example.com:8080",
				HTTPSProxy: "https://proxy.example.com:8443",
				FTPProxy:   "ftp://proxy.example.com:21",
				NoProxy:    "localhost,127.0.0.1,.internal",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ToCLIProxyConfig(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestToCLIProxyConfigs(t *testing.T) {
	t.Helper()
	tests := []struct {
		name     string
		input    map[string]config.ProxyConfig
		expected map[string]configfile.ProxyConfig
	}{
		{
			name:     "empty map",
			input:    map[string]config.ProxyConfig{},
			expected: map[string]configfile.ProxyConfig{},
		},
		{
			name: "single proxy config",
			input: map[string]config.ProxyConfig{
				"default": {
					HTTPProxy:  "http://proxy.example.com:8080",
					HTTPSProxy: "https://proxy.example.com:8443",
				},
			},
			expected: map[string]configfile.ProxyConfig{
				"default": {
					HTTPProxy:  "http://proxy.example.com:8080",
					HTTPSProxy: "https://proxy.example.com:8443",
				},
			},
		},
		{
			name: "multiple proxy configs",
			input: map[string]config.ProxyConfig{
				"development": {
					HTTPProxy: "http://dev-proxy.example.com:8080",
					NoProxy:   "localhost",
				},
				"production": {
					HTTPSProxy: "https://prod-proxy.example.com:8443",
					FTPProxy:   "ftp://prod-proxy.example.com:21",
					NoProxy:    "localhost,127.0.0.1,.internal",
				},
			},
			expected: map[string]configfile.ProxyConfig{
				"development": {
					HTTPProxy: "http://dev-proxy.example.com:8080",
					NoProxy:   "localhost",
				},
				"production": {
					HTTPSProxy: "https://prod-proxy.example.com:8443",
					FTPProxy:   "ftp://prod-proxy.example.com:21",
					NoProxy:    "localhost,127.0.0.1,.internal",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ToCLIProxyConfigs(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
