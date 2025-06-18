package client

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"
)

// NetworkConnect connects a container to a network
func (c *Client) NetworkConnect(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.NetworkConnect(ctx, networkID, containerID, config)
}

// NetworkCreate creates a new network
func (c *Client) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return network.CreateResponse{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.NetworkCreate(ctx, name, options)
}

// NetworkInspect inspects a network
func (c *Client) NetworkInspect(ctx context.Context, name string, options network.InspectOptions) (network.Inspect, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return network.Inspect{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.NetworkInspect(ctx, name, options)
}

// NetworkRemove removes a network
func (c *Client) NetworkRemove(ctx context.Context, name string) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.NetworkRemove(ctx, name)
}
