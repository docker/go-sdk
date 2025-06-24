package image

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
)

// ImageRemoveClient is a client that can remove images.
type ImageRemoveClient interface {
	ImageClient

	// ImageRemove removes an image from the local repository.
	ImageRemove(context.Context, string, image.RemoveOptions) ([]image.DeleteResponse, error)
}

// Remove removes an image from the local repository.
func Remove(ctx context.Context, image string, opts ...RemoveOption) ([]image.DeleteResponse, error) {
	removeOpts := &removeOptions{}
	for _, opt := range opts {
		if err := opt(removeOpts); err != nil {
			return nil, fmt.Errorf("apply remove option: %w", err)
		}
	}

	if image == "" {
		return nil, errors.New("image is required")
	}

	if removeOpts.removeClient == nil {
		removeOpts.removeClient = client.DefaultClient
	}

	resp, err := removeOpts.removeClient.ImageRemove(ctx, image, removeOpts.removeOptions)
	if err != nil {
		return nil, fmt.Errorf("remove image: %w", err)
	}

	return resp, nil
}
