package container

import (
	"context"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// Inspect returns the container's raw info
func (c *Container) Inspect(ctx context.Context) (client.ContainerInspectResult, error) {
	inspect, err := c.dockerClient.ContainerInspect(ctx, c.ID(), client.ContainerInspectOptions{})
	if err != nil {
		return client.ContainerInspectResult{}, err
	}

	return inspect, nil
}

// InspectWithOptions returns the container's raw info, passing custom options.
//
// This method may be deprecated in the near future, to be replaced by functional options for Inspect.
func (c *Container) InspectWithOptions(ctx context.Context, options client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	inspect, err := c.dockerClient.ContainerInspect(ctx, c.ID(), options)
	if err != nil {
		return client.ContainerInspectResult{}, err
	}

	return inspect, nil
}

// State returns container's running state.
func (c *Container) State(ctx context.Context) (*container.State, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return nil, err
	}

	return inspect.Container.State, nil
}
