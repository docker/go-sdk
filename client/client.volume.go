package client

import (
	"context"

	"github.com/moby/moby/client"
)

// VolumeCreate creates a new volume.
func (c *sdkClient) VolumeCreate(ctx context.Context, options client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
	// Add the labels that identify this as a volume created by the SDK.
	AddSDKLabels(options.Labels)

	return c.APIClient.VolumeCreate(ctx, options)
}
