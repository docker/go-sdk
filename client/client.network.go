package client

import (
	"context"

	"github.com/docker/docker/api/types/network"
)

// NetworkConnect connects a container to a network
func (c *Client) NetworkConnect(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error {
	return c.Client().NetworkConnect(ctx, networkID, containerID, config)
}

// NetworkCreate creates a new network
func (c *Client) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	return c.Client().NetworkCreate(ctx, name, options)
}

// NetworkInspect inspects a network
func (c *Client) NetworkInspect(ctx context.Context, name string, options network.InspectOptions) (network.Inspect, error) {
	return c.Client().NetworkInspect(ctx, name, options)
}

// NetworkRemove removes a network
func (c *Client) NetworkRemove(ctx context.Context, name string) error {
	return c.Client().NetworkRemove(ctx, name)
}
