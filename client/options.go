package client

import (
	"context"
	"errors"
	"log/slog"

	"github.com/docker/docker/client"
)

// ClientOption is a type that represents an option for configuring a client.
// It is compatible with docker's Opt type.
type ClientOption interface {
	// Apply applies the option to the client.
	// This method is used to make ClientOption compatible with docker's Opt type.
	Apply(*sdkClient) error
}

// dockerOptAdapter adapts a docker Opt to our ClientOption interface
type dockerOptAdapter struct {
	opt client.Opt
}

// Apply implements the ClientOption interface, adding the docker Opt to the client.
func (a *dockerOptAdapter) Apply(c *sdkClient) error {
	c.dockerOpts = append(c.dockerOpts, a.opt)
	return nil
}

// FromDockerOpt converts a docker Opt to our ClientOption
func FromDockerOpt(opt client.Opt) ClientOption {
	return &dockerOptAdapter{opt: opt}
}

// funcOpt is a function that implements ClientOption
type funcOpt func(*sdkClient) error

// Apply implements the ClientOption interface.
func (f funcOpt) Apply(c *sdkClient) error {
	return f(c)
}

// newClientOption creates a new ClientOption from a function
func newClientOption(f func(*sdkClient) error) ClientOption {
	return funcOpt(f)
}

// WithDockerAPI returns a client option that sets the docker client used to access Docker API.
func WithDockerAPI(api client.APIClient) ClientOption {
	return newClientOption(func(c *sdkClient) error {
		c.APIClient = api
		return nil
	})
}

// WithDockerHost returns a client option that sets the docker host for the client.
func WithDockerHost(dockerHost string) ClientOption {
	return newClientOption(func(c *sdkClient) error {
		c.dockerHost = dockerHost
		return nil
	})
}

// WithDockerContext returns a client option that sets the docker context for the client.
// If set, the client will use the docker context to determine the docker host.
// If used in combination with [WithDockerHost], the host in the context will take precedence.
func WithDockerContext(dockerContext string) ClientOption {
	return newClientOption(func(c *sdkClient) error {
		c.dockerContext = dockerContext
		return nil
	})
}

// WithExtraHeaders returns a client option that sets the extra headers for the client.
func WithExtraHeaders(headers map[string]string) ClientOption {
	return newClientOption(func(c *sdkClient) error {
		c.extraHeaders = headers
		return nil
	})
}

// WithHealthCheck returns a client option that sets the health check for the client.
// If not set, the default health check will be used, which retries the ping to the
// docker daemon until it is ready, three times, or the context is done.
func WithHealthCheck(healthCheck func(ctx context.Context) func(c SDKClient) error) ClientOption {
	return newClientOption(func(c *sdkClient) error {
		if healthCheck == nil {
			return errors.New("health check is nil")
		}

		c.healthCheck = healthCheck
		return nil
	})
}

// WithLogger returns a client option that sets the logger for the client.
func WithLogger(log *slog.Logger) ClientOption {
	return newClientOption(func(c *sdkClient) error {
		c.log = log
		return nil
	})
}
