package client

import (
	"context"
	"fmt"
)

// VolumeRemove removes a volume.
func (c *Client) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.VolumeRemove(ctx, volumeID, force)
}
