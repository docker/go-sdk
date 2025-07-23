package client

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
)

// VolumeCreate creates a new volume.
func (c *Client) VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return volume.Volume{}, fmt.Errorf("docker client: %w", err)
	}

	// Add the labels that identify this as a volume created by the SDK.
	AddSDKLabels(options.Labels)

	return dockerClient.VolumeCreate(ctx, options)
}

// VolumeInspect inspects a volume.
func (c *Client) VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return volume.Volume{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.VolumeInspect(ctx, volumeID)
}

// VolumeList lists volumes.
func (c *Client) VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return volume.ListResponse{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.VolumeList(ctx, options)
}

// VolumeRemove removes a volume.
func (c *Client) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.VolumeRemove(ctx, volumeID, force)
}
