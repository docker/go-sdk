package client

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/build"
)

// ImageBuild builds an image from a build context and options.
func (c *sdkClient) ImageBuild(ctx context.Context, context io.Reader, options build.ImageBuildOptions) (build.ImageBuildResponse, error) {
	// Add client labels
	AddSDKLabels(options.Labels)

	return c.APIClient.ImageBuild(ctx, context, options)
}
