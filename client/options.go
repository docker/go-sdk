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
	Apply(*Client) error
}

// dockerOptAdapter adapts a docker Opt to our ClientOption interface
type dockerOptAdapter struct {
	opt client.Opt
}

// Apply implements the ClientOption interface, adding the docker Opt to the client.
func (a *dockerOptAdapter) Apply(c *Client) error {
	c.dockerOpts = append(c.dockerOpts, a.opt)
	return nil
}

// FromDockerOpt converts a docker Opt to our ClientOption
func FromDockerOpt(opt client.Opt) ClientOption {
	return &dockerOptAdapter{opt: opt}
}

// funcOpt is a function that implements ClientOption
type funcOpt func(*Client) error

// Apply implements the ClientOption interface.
func (f funcOpt) Apply(c *Client) error {
	return f(c)
}

// NewClientOption creates a new ClientOption from a function
func NewClientOption(f func(*Client) error) ClientOption {
	return funcOpt(f)
}

// WithDockerHost returns a client option that sets the docker host for the client.
func WithDockerHost(dockerHost string) ClientOption {
	return NewClientOption(func(c *Client) error {
		c.dockerHost = dockerHost
		return nil
	})
}

// WithDockerContext returns a client option that sets the docker context for the client.
// If set, the client will use the docker context to determine the docker host.
// If used in combination with [WithDockerHost], the host in the context will take precedence.
func WithDockerContext(dockerContext string) ClientOption {
	return NewClientOption(func(c *Client) error {
		c.dockerContext = dockerContext
		return nil
	})
}

// WithExtraHeaders returns a client option that sets the extra headers for the client.
func WithExtraHeaders(headers map[string]string) ClientOption {
	return NewClientOption(func(c *Client) error {
		c.extraHeaders = headers
		return nil
	})
}

// WithHealthCheck returns a client option that sets the health check for the client.
// If not set, the default health check will be used, which retries the ping to the
// docker daemon until it is ready, three times, or the context is done.
func WithHealthCheck(healthCheck func(ctx context.Context) func(c *Client) error) ClientOption {
	return NewClientOption(func(c *Client) error {
		if healthCheck == nil {
			return errors.New("health check is nil")
		}

		c.healthCheck = healthCheck
		return nil
	})
}

// WithLogger returns a client option that sets the logger for the client.
func WithLogger(log *slog.Logger) ClientOption {
	return NewClientOption(func(c *Client) error {
		c.log = log
		return nil
	})
}
