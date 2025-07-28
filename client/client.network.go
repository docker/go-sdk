package client

import (
	"context"

	"github.com/moby/moby/api/types/network"
)

// NetworkCreate creates a new network
func (c *Client) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	// Add the labels that identify this as a network created by the SDK.
	AddSDKLabels(options.Labels)

	return c.NetworkCreate(ctx, name, options)
}
