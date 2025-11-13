package client

import (
	"fmt"

	"github.com/docker/docker/api/types/registry"
	cr "github.com/docker/go-sdk/client/registry"
)

// AuthConfigForImage returns the auth config for a single image
func (c *sdkClient) AuthConfigForImage(img string) (string, registry.AuthConfig, error) {
	ref, err := cr.ParseImageRef(img)
	if err != nil {
		return "", registry.AuthConfig{}, fmt.Errorf("parse image ref: %w", err)
	}

	authConfig, err := c.AuthConfigForHostname(ref.Registry)
	if err != nil {
		return ref.Registry, registry.AuthConfig{}, err
	}

	authConfig.ServerAddress = ref.Registry
	return ref.Registry, authConfig, nil
}

func (c *sdkClient) AuthConfigForHostname(host string) (registry.AuthConfig, error) {
	config, err := c.config.GetAuthConfig(host)
	if err != nil {
		return registry.AuthConfig{}, err
	}
	return registry.AuthConfig{
		Username:      config.Username,
		Password:      config.Password,
		IdentityToken: config.IdentityToken,
		RegistryToken: config.RegistryToken,
		Auth:          config.Auth,
		ServerAddress: config.ServerAddress,
	}, nil
}
