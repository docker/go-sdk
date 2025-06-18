package client

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// ImageInspect inspects an image.
func (c *Client) ImageInspect(ctx context.Context, imageID string, inspectOpts ...client.ImageInspectOption) (image.InspectResponse, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return image.InspectResponse{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ImageInspect(ctx, imageID, inspectOpts...)
}

// ImagePull pulls an image from a remote registry.
func (c *Client) ImagePull(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ImagePull(ctx, image, options)
}
