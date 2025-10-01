package client

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

// NetworkCreate creates a new network
func (c *sdkClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	// Add the labels that identify this as a network created by the SDK.
	AddSDKLabels(options.Labels)

	return c.APIClient.NetworkCreate(ctx, name, options)
}

// NetworkDisconnect disconnects a container from a network
func (c *Client) NetworkDisconnect(ctx context.Context, networkID, containerID string, force bool) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.NetworkDisconnect(ctx, networkID, containerID, force)
}

// NetworksPrune deletes unused networks
func (c *Client) NetworksPrune(ctx context.Context, pruneFilters filters.Args) (network.PruneReport, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return network.PruneReport{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.NetworksPrune(ctx, pruneFilters)
}
