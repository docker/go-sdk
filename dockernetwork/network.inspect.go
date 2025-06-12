package dockernetwork

import (
	"context"
	"errors"

	"github.com/docker/docker/api/types/network"
)

type inspectOptions struct {
	cache   bool
	options network.InspectOptions
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
func WithInspectOptions(opts network.InspectOptions) InspectOptions {
	return func(o *inspectOptions) error {
		o.options = opts
		return nil
	}
}

// Inspect inspects the network, caching the results.
func (n *Network) Inspect(ctx context.Context, opts ...InspectOptions) (network.Inspect, error) {
	var zero network.Inspect
	if n.dockerClient == nil {
		return zero, errors.New("docker client is not initialized")
	}

	inspectOptions := &inspectOptions{
		cache: true, // cache the result by default
	}
	for _, opt := range opts {
		if err := opt(inspectOptions); err != nil {
			return zero, err
		}
	}

	if inspectOptions.cache {
		// if the result was already cached, return it
		if n.inspect.ID != "" {
			return n.inspect, nil
		}

		// else, log a warning and inspect the network
		n.dockerClient.Logger().Warn("network not inspected yet, inspecting now", "network", n.ID(), "cache", inspectOptions.cache)
	}

	inspect, err := n.dockerClient.NetworkInspect(ctx, n.ID(), inspectOptions.options)
	if err != nil {
		return zero, err
	}

	// cache the result for subsequent calls
	n.inspect = inspect

	return inspect, nil
}
