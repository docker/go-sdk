package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types/registry"
)

// This is used by the docker CLI in cases where an oauth identity token is used.
// In that case the username is stored literally as `<token>`
// When fetching the credentials we check for this value to determine if.
const tokenUsername = "<token>"

// AuthConfigs returns the auth configs for the given images.
// The images slice must contain images that are used in a Dockerfile.
// The returned map is keyed by the registry registry hostname for each image.
func AuthConfigs(images ...string) (map[string]AuthConfig, error) {
	cfg, err := Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return cfg.AuthConfigsForImages(images)
}

// AuthConfigForHostname gets registry credentials for the passed in registry host.
//
// This will use [Load] to read registry auth details from the config.
// If the config doesn't exist, it will attempt to load registry credentials using the default credential helper for the platform.
func AuthConfigForHostname(hostname string) (AuthConfig, error) {
	cfg, err := Load()
	if err != nil {
		return AuthConfig{}, fmt.Errorf("load config: %w", err)
	}

	return cfg.AuthConfigForHostname(hostname)
}

func (authConfig AuthConfig) EncodeBase64() (string, error) {
	jsonAuth, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(jsonAuth), nil
}

func (authConfig AuthConfig) ToRegistryAuthConfig() registry.AuthConfig {
	return registry.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		Email:         "",
		ServerAddress: authConfig.ServerAddress,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}
}
