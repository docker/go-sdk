package dockercontainer

import (
	"context"

	"github.com/docker/docker/api/types/container"
)

// Inspect returns the container's raw info
func (c *Container) Inspect(ctx context.Context) (*container.InspectResponse, error) {
	inspect, err := c.dockerClient.Client().ContainerInspect(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	return &inspect, nil
}

// State returns container's running state.
func (c *Container) State(ctx context.Context) (*container.State, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return nil, err
	}

	return inspect.State, nil
}
