package dockerclient

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/image"
)

// ImagePull pulls an image from a remote registry.
func (c *Client) ImagePull(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error) {
	return c.client.ImagePull(ctx, image, options)
}
