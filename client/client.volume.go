package client

import (
	"context"

	"github.com/docker/docker/api/types/volume"
)

// VolumeCreate creates a new volume.
func (c *Client) VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	// Add the labels that identify this as a volume created by the SDK.
	AddSDKLabels(options.Labels)

	return c.Client.VolumeCreate(ctx, options)
}
