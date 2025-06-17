package client

import (
	"context"
)

// VolumeRemove removes a volume.
func (c *Client) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return c.Client().VolumeRemove(ctx, volumeID, force)
}
