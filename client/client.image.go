package client

import (
	"context"
	"io"

	"github.com/moby/moby/client"
)

// ImageBuild builds an image from a build context and options.
func (c *sdkClient) ImageBuild(ctx context.Context, context io.Reader, options client.ImageBuildOptions) (client.ImageBuildResult, error) {
	// Add client labels
	AddSDKLabels(options.Labels)

	return c.APIClient.ImageBuild(ctx, context, options)
}
