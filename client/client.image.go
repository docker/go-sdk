package client

import (
	"context"
	"io"

	"github.com/docker/go-sdk/image"
	"github.com/moby/moby/api/types/build"
)

// ImageBuild builds an image from a build context and options.
func (c *Client) ImageBuild(ctx context.Context, context io.Reader, options build.ImageBuildOptions) (build.ImageBuildResponse, error) {
	// Add client labels
	AddSDKLabels(options.Labels)

	return c.APIClient.ImageBuild(ctx, options.Context, options)
}

func (c *Client) Pull(ctx context.Context, imageID string, opts ...image.PullOption) (string, error) {
	// TODO
	return "digest", nil
}
