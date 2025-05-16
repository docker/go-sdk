package dockerconfig

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"runtime"
	"strings"
)

// This is used by the docker CLI in cases where an oauth identity token is used.
// In that case the username is stored literally as `<token>`
// When fetching the credentials we check for this value to determine if.
const tokenUsername = "<token>"

// RegistryCredentialsForHostname gets registry credentials for the passed in registry host.
//
// This will use [Load] to read registry auth details from the config.
// If the config doesn't exist, it will attempt to load registry credentials using the default credential helper for the platform.
func RegistryCredentialsForHostname(hostname string) (string, string, error) {
	cfg, err := Load()
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return "", "", fmt.Errorf("load default config: %w", err)
		}

		return credentialsFromHelper("", hostname)
	}

	return cfg.RegistryCredentialsForHostname(hostname)
}

// ResolveRegistryHost can be used to transform a docker registry host name into what is used for the docker config/cred helpers
//
// This is useful for using with containerd authorizers.
// Naturally this only transforms docker hub URLs.
func ResolveRegistryHost(host string) string {
	switch host {
	case "index.docker.io", "docker.io", "https://index.docker.io/v1/", "registry-1.docker.io":
		return "https://index.docker.io/v1/"
	}
	return host
}

// RegistryCredentialsForHostname gets credentials, if any, for the provided hostname.
//
// Hostnames should already be resolved using [ResolveRegistryHost].
//
// If the returned username string is empty, the password is an identity token.
func (c *Config) RegistryCredentialsForHostname(hostname string) (string, string, error) {
	h, ok := c.CredentialHelpers[hostname]
	if ok {
		return credentialsFromHelper(h, hostname)
	}

	if c.CredentialsStore != "" {
		username, password, err := credentialsFromHelper(c.CredentialsStore, hostname)
		if err != nil {
			return "", "", fmt.Errorf("get credentials from store: %w", err)
		}

		if username != "" || password != "" {
			return username, password, nil
		}
	}

	auth, ok := c.AuthConfigs[hostname]
	if !ok {
		return credentialsFromHelper("", hostname)
	}

	if auth.IdentityToken != "" {
		return "", auth.IdentityToken, nil
	}

	if auth.Username != "" && auth.Password != "" {
		return auth.Username, auth.Password, nil
	}

	return decodeBase64Auth(auth)
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
