package config

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-sdk/config/auth"
)

var cacheInitMutex sync.Mutex

// authConfigCache holds the caching state for a Config instance
type authConfigCache struct {
	entries map[string]AuthConfig
	mutex   sync.RWMutex
	key     string
}

// clearAuthCache clears the cached auth configs
func (c *Config) clearAuthCache() {
	cache := c.getCache()
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	cache.entries = make(map[string]AuthConfig)
}

// cacheStats returns statistics about the auth config cache
func (c *Config) cacheStats() cacheStats {
	cache := c.getCache()
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	return cacheStats{
		Size:     len(cache.entries),
		CacheKey: cache.key,
	}
}

type cacheStats struct {
	Size     int
	CacheKey string
}

// getCache safely returns the cache, initializing it if necessary
func (c *Config) getCache() *authConfigCache {
	c.initCache()
	return c.cache.Load().(*authConfigCache)
}

// initCache initializes the cache if it hasn't been initialized yet
func (c *Config) initCache() {
	// Try to load existing cache
	if c.cache.Load() != nil {
		return // Fast path - cache already initialized
	}

	cacheInitMutex.Lock()
	defer cacheInitMutex.Unlock()

	// Double-check pattern
	if c.cache.Load() != nil {
		return // Another goroutine initialized it
	}

	newCache := &authConfigCache{
		entries: make(map[string]AuthConfig),
		key:     c.generateCacheKey(),
	}

	c.cache.Store(newCache)
}

// generateCacheKey creates a unique key for this config instance
func (c *Config) generateCacheKey() string {
	h := md5.New()
	if err := json.NewEncoder(h).Encode(c); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(h.Sum(nil))
}

// AuthConfigForHostname returns the auth config for the given hostname with caching
func (c *Config) AuthConfigForHostname(hostname string) (AuthConfig, error) {
	cache := c.getCache()

	// Try cache first
	cache.mutex.RLock()
	if authConfig, exists := cache.entries[hostname]; exists {
		cache.mutex.RUnlock()
		return authConfig, nil
	}
	cache.mutex.RUnlock()

	// Cache miss - resolve auth config
	authConfig, err := c.resolveAuthConfigForHostname(hostname)
	if err != nil {
		return AuthConfig{}, err
	}

	// Cache the result
	cache.mutex.Lock()
	cache.entries[hostname] = authConfig
	cache.mutex.Unlock()

	return authConfig, nil
}

// AuthConfigsForImages returns auth configs for multiple images with caching
func (c *Config) AuthConfigsForImages(images []string) (map[string]AuthConfig, error) {
	result := make(map[string]AuthConfig)
	var errs []error

	// Process each image
	for _, image := range images {
		registry, authConfig, err := c.AuthConfigForImage(image)
		if err != nil {
			if !errors.Is(err, ErrCredentialsNotFound) {
				errs = append(errs, fmt.Errorf("auth config for %q: %w", registry, err))
				continue
			}
			// Skip if credentials not found
			continue
		}

		authConfig.ServerAddress = registry
		result[registry] = authConfig
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return result, nil
}

// AuthConfigForImage returns the auth config for a single image
func (c *Config) AuthConfigForImage(image string) (string, AuthConfig, error) {
	ref, err := auth.ParseImageRef(image)
	if err != nil {
		return "", AuthConfig{}, fmt.Errorf("parse image ref: %w", err)
	}

	authConfig, err := c.AuthConfigForHostname(ref.Registry)
	if err != nil {
		return ref.Registry, AuthConfig{}, err
	}

	authConfig.ServerAddress = ref.Registry
	return ref.Registry, authConfig, nil
}

// ParseProxyConfig computes proxy configuration by retrieving the config for the provided host and
// then checking this against any environment variables provided to the container
func (c *Config) ParseProxyConfig(host string, runOpts map[string]*string) map[string]*string {
	var cfgKey string

	if _, ok := c.Proxies[host]; !ok {
		cfgKey = "default"
	} else {
		cfgKey = host
	}

	conf := c.Proxies[cfgKey]
	permitted := map[string]*string{
		"HTTP_PROXY":  &conf.HTTPProxy,
		"HTTPS_PROXY": &conf.HTTPSProxy,
		"NO_PROXY":    &conf.NoProxy,
		"FTP_PROXY":   &conf.FTPProxy,
		"ALL_PROXY":   &conf.AllProxy,
	}
	m := runOpts
	if m == nil {
		m = make(map[string]*string)
	}
	for k := range permitted {
		if *permitted[k] == "" {
			continue
		}
		if _, ok := m[k]; !ok {
			m[k] = permitted[k]
		}
		if _, ok := m[strings.ToLower(k)]; !ok {
			m[strings.ToLower(k)] = permitted[k]
		}
	}
	return m
}

// Save saves the config to the file system
func (c *Config) Save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	return os.WriteFile(c.filepath, data, 0o644)
}

// resolveAuthConfigForHostname performs the actual auth config resolution
func (c *Config) resolveAuthConfigForHostname(hostname string) (AuthConfig, error) {
	// Check credential helpers first
	if helper, exists := c.CredentialHelpers[hostname]; exists {
		return c.resolveFromCredentialHelper(helper, hostname)
	}

	// Check global credential store
	if c.CredentialsStore != "" {
		if authConfig, err := c.resolveFromCredentialHelper(c.CredentialsStore, hostname); err == nil {
			if authConfig.Username != "" || authConfig.Password != "" {
				return authConfig, nil
			}
		}
	}

	// Check stored auth configs
	if authConfig, exists := c.AuthConfigs[hostname]; exists {
		return c.processStoredAuthConfig(authConfig, hostname)
	}

	// Fallback to default credential helper
	return c.resolveFromCredentialHelper("", hostname)
}

// resolveFromCredentialHelper resolves credentials from a credential helper
func (c *Config) resolveFromCredentialHelper(helper, hostname string) (AuthConfig, error) {
	// Use existing credentialsFromHelper function but adapt to return AuthConfig
	credentials, err := credentialsFromHelper(helper, hostname)
	if err != nil {
		return AuthConfig{}, err
	}

	return credentials, nil
}

// processStoredAuthConfig processes auth config from stored configuration
func (c *Config) processStoredAuthConfig(stored AuthConfig, hostname string) (AuthConfig, error) {
	authConfig := AuthConfig{
		Auth:          stored.Auth,
		IdentityToken: stored.IdentityToken,
		Password:      stored.Password,
		RegistryToken: stored.RegistryToken,
		ServerAddress: hostname,
		Username:      stored.Username,
	}

	// Handle different auth scenarios
	switch {
	case authConfig.IdentityToken != "":
		// Identity token case
		authConfig.Username = ""
		authConfig.Password = authConfig.IdentityToken

	case authConfig.Username != "" && authConfig.Password != "":
		// Username/password case - already set

	case authConfig.Auth != "":
		// Base64 auth case
		user, pass, err := decodeBase64Auth(authConfig)
		if err != nil {
			return AuthConfig{}, fmt.Errorf("decode base64 auth: %w", err)
		}
		authConfig.Username = user
		authConfig.Password = pass

	default:
		// No stored credentials, try credential helper
		return c.resolveFromCredentialHelper("", hostname)
	}

	return authConfig, nil
}

// decodeBase64Auth decodes the legacy file-based auth storage from the docker CLI.
// It takes the "Auth" filed from AuthConfig and decodes that into a username and password.
//
// If "Auth" is empty, an empty user/pass will be returned, but not an error.
func decodeBase64Auth(auth AuthConfig) (string, string, error) {
	if auth.Auth == "" {
		return "", "", nil
	}

	decLen := base64.StdEncoding.DecodedLen(len(auth.Auth))
	decoded := make([]byte, decLen)
	n, err := base64.StdEncoding.Decode(decoded, []byte(auth.Auth))
	if err != nil {
		return "", "", fmt.Errorf("decode auth: %w", err)
	}

	decoded = decoded[:n]

	const sep = ":"
	user, pass, found := strings.Cut(string(decoded), sep)
	if !found {
		return "", "", fmt.Errorf("invalid auth: missing %q separator", sep)
	}

	return user, pass, nil
}
