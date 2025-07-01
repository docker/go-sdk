package config

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/docker/go-sdk/config/auth"
)

// authConfigResult is a result looking up auth details for key.
type authConfigResult struct {
	key string
	cfg AuthConfig
	err error
}

// credentialsCache is a cache for registry credentials.
type credentialsCache struct {
	entries map[string]credentials
	mtx     sync.RWMutex
}

// credentials represents the username and password for a registry.
type credentials struct {
	username string
	password string
}

// creds is the global credentials cache.
var creds = &credentialsCache{entries: map[string]credentials{}}

// AuthConfig updates the details in authConfig for the given hostname
// as determined by the details in configKey.
func (c *credentialsCache) AuthConfig(hostname, configKey string, authConfig *AuthConfig) error {
	u, p, err := creds.get(hostname, configKey)
	if err != nil {
		return err
	}

	if u != "" {
		authConfig.Username = u
		authConfig.Password = p
	} else {
		authConfig.IdentityToken = p
	}

	return nil
}

// get returns the username and password for the given hostname
// as determined by the details in configPath.
// If the username is empty, the password is an identity token.
func (c *credentialsCache) get(hostname, configKey string) (string, string, error) {
	key := configKey + ":" + hostname
	c.mtx.RLock()
	entry, ok := c.entries[key]
	c.mtx.RUnlock()

	if ok {
		return entry.username, entry.password, nil
	}

	// No entry found, request and cache.
	cfg, err := RegistryCredentialsForHostname(hostname)
	if err != nil {
		return "", "", fmt.Errorf("getting credentials for %s: %w", hostname, err)
	}

	c.mtx.Lock()
	c.entries[key] = credentials{username: cfg.Username, password: cfg.Password}
	c.mtx.Unlock()

	return cfg.Username, cfg.Password, nil
}

// authConfigs returns the auth configs for the current Docker config.
//
// It uses [Load] to read registry auth details from the config.
// If the config doesn't exist, it attempts to load registry credentials using
// the default credential helper for the platform.
//
// The returned map is keyed by the registry hostname.
func authConfigs() (map[string]AuthConfig, error) {
	cfg, err := Load()
	if err != nil {
		return nil, fmt.Errorf("load default config: %w", err)
	}

	key, err := configKey(&cfg)
	if err != nil {
		return nil, err
	}

	size := len(cfg.AuthConfigs) + len(cfg.CredentialHelpers)
	cfgs := make(map[string]AuthConfig, size)
	results := make(chan authConfigResult, size)

	var wg sync.WaitGroup
	wg.Add(size)
	for k, v := range cfg.AuthConfigs {
		go func(k string, v AuthConfig) {
			defer wg.Done()

			ac := AuthConfig{
				Auth:          v.Auth,
				Email:         v.Email,
				IdentityToken: v.IdentityToken,
				Password:      v.Password,
				RegistryToken: v.RegistryToken,
				ServerAddress: v.ServerAddress,
				Username:      v.Username,
			}

			switch {
			case ac.Username == "" && ac.Password == "":
				// Look up credentials from the credential store.
				if err := creds.AuthConfig(k, key, &ac); err != nil {
					results <- authConfigResult{err: err}
					return
				}
			case ac.Auth == "":
				// Create auth from the username and password encoding.
				ac.Auth = base64.StdEncoding.EncodeToString([]byte(ac.Username + ":" + ac.Password))
			}

			results <- authConfigResult{key: k, cfg: ac}
		}(k, v)
	}

	// In the case where the auth field in the .docker/conf.json is empty, and the user has
	// credential helpers registered the auth comes from there.
	for k := range cfg.CredentialHelpers {
		go func(k string) {
			defer wg.Done()

			var ac AuthConfig
			if err := creds.AuthConfig(k, key, &ac); err != nil {
				results <- authConfigResult{err: err}
				return
			}

			results <- authConfigResult{key: k, cfg: ac}
		}(k)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var errs []error
	for result := range results {
		if result.err != nil {
			errs = append(errs, result.err)
			continue
		}

		cfgs[result.key] = result.cfg
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return cfgs, nil
}

// dockerImageAuth returns the auth config for the given Docker image.
func dockerImageAuth(image string, configs map[string]AuthConfig) (string, AuthConfig, error) {
	ref, err := auth.ParseImageRef(image)
	if err != nil {
		return "", AuthConfig{}, fmt.Errorf("parse image ref: %w", err)
	}

	if cfg, ok := registryAuth(ref.Registry, configs); ok {
		return ref.Registry, cfg, nil
	}

	return ref.Registry, AuthConfig{}, ErrCredentialsNotFound
}

// registryAuth returns the auth config for the given registry.
// If the registry is not found, it returns false.
func registryAuth(reg string, cfgs map[string]AuthConfig) (AuthConfig, bool) {
	if cfg, ok := cfgs[reg]; ok {
		return cfg, true
	}

	// fallback match using authentication key host
	for k, cfg := range cfgs {
		keyURL, err := url.Parse(k)
		if err != nil {
			continue
		}

		host := keyURL.Host
		if keyURL.Scheme == "" {
			// url.Parse: The url may be relative (a path, without a host) [...]
			host = keyURL.Path
		}

		if host == reg {
			return cfg, true
		}
	}

	return AuthConfig{}, false
}

// configKey returns a key to use for caching credentials based on
// the contents of the currently active config.
func configKey(cfg *Config) (string, error) {
	h := md5.New()
	if err := json.NewEncoder(h).Encode(cfg); err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
