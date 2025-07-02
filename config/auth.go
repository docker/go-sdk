package config

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// This is used by the docker CLI in cases where an oauth identity token is used.
// In that case the username is stored literally as `<token>`
// When fetching the credentials we check for this value to determine if.
const tokenUsername = "<token>"

// AuthConfigs returns the auth configs for the given images.
// The images slice must contain images that are used in a Dockerfile. You can use:
// - [ImagesFromDockerfile] to extract images from a Dockerfile path.
// - [ImagesFromTarReader] to extract images from a tar reader.
// - [ImagesFromReader] to extract images from a reader.
//
// The returned map is keyed by the registry registry hostname for each image.
func AuthConfigs(images ...string) (map[string]AuthConfig, error) {
	cfg, err := Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return cfg.AuthConfigsForImages(images)
}

// RegistryCredentialsForHostname gets registry credentials for the passed in registry host.
//
// This will use [Load] to read registry auth details from the config.
// If the config doesn't exist, it will attempt to load registry credentials using the default credential helper for the platform.
func RegistryCredentialsForHostname(hostname string) (AuthConfig, error) {
	cfg, err := Load()
	if err != nil {
		return AuthConfig{}, fmt.Errorf("load config: %w", err)
	}

	return cfg.AuthConfigForHostname(hostname)
}

// RegistryCredentialsForHostname gets credentials, if any, for the provided hostname.
//
// Hostnames should already be resolved using [ResolveRegistryHost].
//
// If the returned username string is empty, the password is an identity token.
func (c *Config) RegistryCredentialsForHostname(hostname string) (AuthConfig, error) {
	var zero AuthConfig
	h, ok := c.CredentialHelpers[hostname]
	if ok {
		return credentialsFromHelper(h, hostname)
	}

	if c.CredentialsStore != "" {
		creds, err := credentialsFromHelper(c.CredentialsStore, hostname)
		if err != nil {
			return zero, fmt.Errorf("get credentials from store: %w", err)
		}

		if creds.Username != "" || creds.Password != "" {
			return creds, nil
		}
	}

	authConfig, ok := c.AuthConfigs[hostname]
	if !ok {
		return credentialsFromHelper("", hostname)
	}

	creds := AuthConfig{}

	if authConfig.IdentityToken != "" {
		creds.Username = ""
		creds.Password = authConfig.IdentityToken
		creds.ServerAddress = hostname
		return creds, nil
	}

	if authConfig.Username != "" && authConfig.Password != "" {
		creds.Username = authConfig.Username
		creds.Password = authConfig.Password
		creds.ServerAddress = hostname
		return creds, nil
	}

	user, pass, err := decodeBase64Auth(authConfig)
	if err != nil {
		return zero, fmt.Errorf("decode base64 auth: %w", err)
	}

	creds.Username = user
	creds.Password = pass
	creds.ServerAddress = hostname

	return creds, nil
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
