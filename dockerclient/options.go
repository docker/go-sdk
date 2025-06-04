package dockerclient

import (
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

// Apply implements the ClientOption interface.
func (a *dockerOptAdapter) Apply(c *Client) error {
	return a.opt(c.client)
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

// WithExtraHeaders returns a client option that sets the extra headers for the client.
func WithExtraHeaders(headers map[string]string) ClientOption {
	return NewClientOption(func(c *Client) error {
		c.extraHeaders = headers
		return nil
	})
}

// WithLogger returns a client option that sets the logger for the client.
func WithLogger(log slog.Logger) ClientOption {
	return NewClientOption(func(c *Client) error {
		c.log = log
		return nil
	})
}
