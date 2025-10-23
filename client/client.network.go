package client

import (
	"context"

	"github.com/moby/moby/client"
)

// NetworkCreate creates a new network
func (c *sdkClient) NetworkCreate(ctx context.Context, name string, options client.NetworkCreateOptions) (client.NetworkCreateResult, error) {
	// Add the labels that identify this as a network created by the SDK.
	AddSDKLabels(options.Labels)

	return c.APIClient.NetworkCreate(ctx, name, options)
}
