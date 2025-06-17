package client

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// ImageInspect inspects an image.
func (c *Client) ImageInspect(ctx context.Context, imageID string, inspectOpts ...client.ImageInspectOption) (image.InspectResponse, error) {
	return c.Client().ImageInspect(ctx, imageID, inspectOpts...)
}

// ImagePull pulls an image from a remote registry.
func (c *Client) ImagePull(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error) {
	return c.Client().ImagePull(ctx, image, options)
}
