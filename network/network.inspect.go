package network

import (
	"context"
	"errors"

	"github.com/moby/moby/client"
)

type inspectOptions struct {
	cache   bool
	options client.NetworkInspectOptions
}

// InspectOptions is a function that modifies the inspect options.
type InspectOptions func(opts *inspectOptions) error

// WithNoCache returns an InspectOptions that disables caching the result of the inspection.
// If passed, the Docker daemon will be queried for the latest information, so it can be
// used for refreshing the cached result of a previous inspection.
func WithNoCache() InspectOptions {
	return func(o *inspectOptions) error {
		o.cache = false
		return nil
	}
}

// WithInspectOptions returns an InspectOptions that sets the inspect options.
func WithInspectOptions(opts client.NetworkInspectOptions) InspectOptions {
	return func(o *inspectOptions) error {
		o.options = opts
		return nil
	}
}

// Inspect inspects the network, caching the results.
func (n *Network) Inspect(ctx context.Context, opts ...InspectOptions) (client.NetworkInspectResult, error) {
	if n.dockerClient == nil {
		return client.NetworkInspectResult{}, errors.New("docker client is not initialized")
	}

	inspectOptions := &inspectOptions{
		cache: true, // cache the result by default
	}
	for _, opt := range opts {
		if err := opt(inspectOptions); err != nil {
			return client.NetworkInspectResult{}, err
		}
	}

	if inspectOptions.cache {
		// if the result was already cached, return it
		if n.inspect.Network.ID != "" {
			return n.inspect, nil
		}

		// else, log a warning and inspect the network
		n.dockerClient.Logger().Warn("network not inspected yet, inspecting now", "network", n.ID(), "cache", inspectOptions.cache)
	}

	inspect, err := n.dockerClient.NetworkInspect(ctx, n.ID(), inspectOptions.options)
	if err != nil {
		return client.NetworkInspectResult{}, err
	}

	// cache the result for subsequent calls
	n.inspect = inspect

	return inspect, nil
}
