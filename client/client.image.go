package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// ImageBuild builds an image from a build context and options.
func (c *Client) ImageBuild(ctx context.Context, options build.ImageBuildOptions) (build.ImageBuildResponse, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return build.ImageBuildResponse{}, fmt.Errorf("docker client: %w", err)
	}

	if options.Context == nil {
		return build.ImageBuildResponse{}, errors.New("build context is nil")
	}

	// Add client labels
	AddSDKLabels(options.Labels)

	return dockerClient.ImageBuild(ctx, options.Context, options)
}

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

// ImageRemove removes an image from the local repository.
func (c *Client) ImageRemove(ctx context.Context, image string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ImageRemove(ctx, image, options)
}

// ImageSave saves an image to a file.
func (c *Client) ImageSave(ctx context.Context, images []string, saveOptions ...client.ImageSaveOption) (io.ReadCloser, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ImageSave(ctx, images, saveOptions...)
}
