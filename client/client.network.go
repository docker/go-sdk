package client

import (
	"context"

	"github.com/docker/docker/api/types/network"
)

// NetworkCreate creates a new network
func (c *sdkClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	// Add the labels that identify this as a network created by the SDK.
	AddSDKLabels(options.Labels)

	return c.APIClient.NetworkCreate(ctx, name, options)
}
