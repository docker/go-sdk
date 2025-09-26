package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig_AuthConfigsForImages(t *testing.T) {
	config := Config{
		AuthConfigs: map[string]AuthConfig{
			"registry1.io": {Username: "user1", Password: "pass1"},
			"registry2.io": {Username: "user2", Password: "pass2"},
		},
	}

	images := []string{
		"registry1.io/repo/image:tag",
		"registry2.io/repo/image:tag",
	}

	authConfigs, err := config.AuthConfigsForImages(images)
	require.NoError(t, err)
	require.Len(t, authConfigs, 2)
	require.Equal(t, "user1", authConfigs["registry1.io"].Username)
	require.Equal(t, "user2", authConfigs["registry2.io"].Username)

	// Verify caching worked
	stats := config.cacheStats()
	require.Equal(t, 2, stats.Size)
}

func TestConfig_CacheManagement(t *testing.T) {
	config := Config{
		AuthConfigs: map[string]AuthConfig{
			"test.io": {Username: "user", Password: "pass"},
		},
	}

	t.Run("cache-initialization", func(t *testing.T) {
		stats := config.cacheStats()
		require.Equal(t, 0, stats.Size)
		require.NotEmpty(t, stats.CacheKey)
	})

	t.Run("cache-population", func(t *testing.T) {
		_, err := config.AuthConfigForHostname("test.io")
		require.NoError(t, err)

		stats := config.cacheStats()
		require.Equal(t, 1, stats.Size)
	})

	t.Run("cache-clearing", func(t *testing.T) {
		config.clearAuthCache()
		stats := config.cacheStats()
		require.Equal(t, 0, stats.Size)
	})
}

func TestConfig_ConcurrentAccess(t *testing.T) {
	config := Config{
		AuthConfigs: map[string]AuthConfig{
			"test.io": {Username: "user", Password: "pass"},
		},
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := config.AuthConfigForHostname("test.io")
			require.NoError(t, err)
		}()
	}

	wg.Wait()
	stats := config.cacheStats()
	require.Equal(t, 1, stats.Size)
}

func TestConfig_CacheKeyGeneration(t *testing.T) {
	config1 := Config{
		AuthConfigs: map[string]AuthConfig{
			"test.io": {Username: "user1", Password: "pass1"},
		},
	}

	config2 := Config{
		AuthConfigs: map[string]AuthConfig{
			"test.io": {Username: "user2", Password: "pass2"},
		},
	}

	stats1 := config1.cacheStats()
	stats2 := config2.cacheStats()

	require.NotEqual(t, stats1.CacheKey, stats2.CacheKey)
}

func TestConfigSave(t *testing.T) {
	tmpDir := t.TempDir()
	setupHome(t, tmpDir)

	dockerDir := filepath.Join(tmpDir, ".docker")

	err := os.MkdirAll(dockerDir, 0o755)
	require.NoError(t, err)

	_, err = os.Create(filepath.Join(dockerDir, FileName))
	require.NoError(t, err)

	c := Config{
		filepath:       filepath.Join(dockerDir, FileName),
		CurrentContext: "test",
		AuthConfigs:    map[string]AuthConfig{},
	}

	require.NoError(t, c.Save())

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, c.CurrentContext, cfg.CurrentContext)
	require.Equal(t, c.AuthConfigs, cfg.AuthConfigs)
}

func TestConfig_AuthConfigForHostname_URLPrefixes(t *testing.T) {
	config := Config{
		AuthConfigs: map[string]AuthConfig{
			"https://index.docker.io/v1/":  {Username: "dockeruser", Password: "dockerpass"},
			"https://registry.example.com": {Username: "exampleuser", Password: "examplepass"},
			"http://unsecure.registry.com": {Username: "unsecureuser", Password: "unsecurepass"},
		},
	}

	t.Run("https prefix should resolve to docker hub", func(t *testing.T) {
		authConfig, err := config.AuthConfigForHostname("https://index.docker.io/v1/")
		require.NoError(t, err)
		require.Equal(t, "dockeruser", authConfig.Username)
		require.Equal(t, "dockerpass", authConfig.Password)
	})

	t.Run("https prefix should be stripped and found", func(t *testing.T) {
		authConfig, err := config.AuthConfigForHostname("https://registry.example.com")
		require.NoError(t, err)
		require.Equal(t, "exampleuser", authConfig.Username)
		require.Equal(t, "examplepass", authConfig.Password)
	})

	t.Run("http prefix should be stripped and found", func(t *testing.T) {
		authConfig, err := config.AuthConfigForHostname("http://unsecure.registry.com")
		require.NoError(t, err)
		require.Equal(t, "unsecureuser", authConfig.Username)
		require.Equal(t, "unsecurepass", authConfig.Password)
	})

	t.Run("hostname without prefix should resolve to stored config with prefix", func(t *testing.T) {
		authConfig, err := config.AuthConfigForHostname("registry.example.com")
		require.NoError(t, err)
		require.Equal(t, "exampleuser", authConfig.Username)
		require.Equal(t, "examplepass", authConfig.Password)
	})

	t.Run("docker.io variants should resolve to docker hub config", func(t *testing.T) {
		authConfig, err := config.AuthConfigForHostname("docker.io")
		require.NoError(t, err)
		require.Equal(t, "dockeruser", authConfig.Username)
		require.Equal(t, "dockerpass", authConfig.Password)

		authConfig2, err := config.AuthConfigForHostname("index.docker.io")
		require.NoError(t, err)
		require.Equal(t, "dockeruser", authConfig2.Username)
		require.Equal(t, "dockerpass", authConfig2.Password)
	})
}
