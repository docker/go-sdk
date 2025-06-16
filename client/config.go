package client

import (
	"errors"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
)

// config represents the configuration for the Docker client.
// User values are read from the specified environment variables.
type config struct {
	// Host is the address of the Docker daemon.
	// Default: ""
	Host string `env:"DOCKER_HOST"`

	// TLSVerify is a flag to enable or disable TLS verification when connecting to a Docker daemon.
	// Default: 0
	TLSVerify bool `env:"DOCKER_TLS_VERIFY"`

	// CertPath is the path to the directory containing the Docker certificates.
	// This is used when connecting to a Docker daemon over TLS.
	// Default: ""
	CertPath string `env:"DOCKER_CERT_PATH"`
}

// newConfig returns a new configuration loaded from the properties file
// located in the user's home directory and overridden by environment variables.
func newConfig(host string) (*config, error) {
	cfg := &config{
		Host: host,
	}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return cfg, nil
}

// validate verifies the configuration is valid.
func (c *config) validate() error {
	if c.TLSVerify && c.CertPath == "" {
		return errors.New("cert path required when TLS is enabled")
	}

	if c.TLSVerify {
		if _, err := os.Stat(c.CertPath); os.IsNotExist(err) {
			return fmt.Errorf("cert path does not exist: %s", c.CertPath)
		}
	}

	if c.Host == "" {
		return errors.New("host is required")
	}

	return nil
}
